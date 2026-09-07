package upgradebot

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"java-app/pkg/network/protocol"
)

type fakeInj struct {
	mu   sync.Mutex
	msgs []*protocol.Message
}

func (f *fakeInj) Inject(msg *protocol.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := protocol.NewMessageWithData(msg.Command, append([]byte(nil), msg.GetData()...))
	f.msgs = append(f.msgs, cp)
	return nil
}

func TestStartRejectedWhenNotInGame(t *testing.T) {
	c := NewController(&fakeInj{})
	err := c.Start(Config{ItemID: 1, TargetLevel: 10, DelayMs: 1})
	if err != "not in game" {
		t.Fatalf("got %q", err)
	}
}

func TestStartRejectedWhenItemMissing(t *testing.T) {
	inj := &fakeInj{}
	c := NewController(inj)
	c.SetSnapshot(Snapshot{InGame: true})
	err := c.Start(Config{ItemID: 1, ItemType: 0, TargetLevel: 10, DelayMs: 1})
	if err != "no inventory" {
		t.Fatalf("got %q", err)
	}
	if c.Running() {
		t.Fatal("must not run without the target item")
	}
	inj.mu.Lock()
	n := len(inj.msgs)
	inj.mu.Unlock()
	if n != 0 {
		t.Fatalf("injected %d packets without inventory", n)
	}
}

func TestStartRejectedWhenOutOfClover(t *testing.T) {
	inj := &fakeInj{}
	c := NewController(inj)
	c.SetSnapshot(Snapshot{
		InGame: true,
		Items:  []*Item{{ID: 1, Type: 0, Name: "Kiem", Level: 6, Quantity: 1}},
	})
	err := c.Start(Config{
		ItemID: 1, ItemType: 0, TargetLevel: 10,
		Clover3Min: 4, Clover3Max: 7,
		DelayMs: 1, StopIfOutOfClover: true,
	})
	if err == "" {
		t.Fatal("expected clover error")
	}
	if c.Running() {
		t.Fatal("must not run without required clover")
	}
	inj.mu.Lock()
	n := len(inj.msgs)
	inj.mu.Unlock()
	if n != 0 {
		t.Fatalf("injected %d packets without clover", n)
	}
}

func TestLoopInjectsSelectCurrencyUpgrade(t *testing.T) {
	inj := &fakeInj{}
	c := NewController(inj)
	c.SetSnapshot(Snapshot{
		InGame: true,
		Items: []*Item{
			{ID: 1, Type: 0, Name: "Kiem", Level: 6, Quantity: 1},
			{ID: 9, Type: 7, Name: "Co 3 la", Quantity: 5, IsClover3: true},
		},
	})
	if err := c.Start(Config{
		ItemID: 1, ItemType: 0, TargetLevel: 7,
		Clover3Min: 4, Clover3Max: 7,
		DelayMs: 5, StopIfOutOfClover: true,
	}); err != "" {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		inj.mu.Lock()
		n := len(inj.msgs)
		inj.mu.Unlock()
		if n >= 4 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.Stop()
	inj.mu.Lock()
	defer inj.mu.Unlock()
	if len(inj.msgs) < 4 {
		t.Fatalf("want select+clover+currency+upgrade, got %d", len(inj.msgs))
	}
	want := []byte{ActionSelect, ActionSelect, ActionCurrency, ActionUpgrade}
	for i := 0; i < 4; i++ {
		b, _ := inj.msgs[i].ReadByte()
		if b != want[i] {
			t.Fatalf("step %d action=%d want %d", i, b, want[i])
		}
	}
}

func TestStopConditionStopsRunning(t *testing.T) {
	c := NewController(&fakeInj{})
	c.SetSnapshot(Snapshot{InGame: true, Items: []*Item{{ID: 1, Level: 6, Quantity: 1}}})
	_ = c.Start(Config{ItemID: 1, TargetLevel: 10, DelayMs: 50})
	msg := protocol.NewMessage(25)
	msg.WriteUTF("Không đủ xu")
	c.OnMessage(msg)
	if c.Running() {
		t.Fatal("should stop")
	}
}

func TestStatusJSONShape(t *testing.T) {
	c := NewController(&fakeInj{})
	c.SetSnapshot(Snapshot{InGame: true})
	raw := c.StatusJSON()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"running", "connected", "in_game", "attempts", "success", "fail", "item_level", "logs"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing %s", k)
		}
	}
}

func statusInt(t *testing.T, c *Controller, key string) int {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(c.StatusJSON()), &m); err != nil {
		t.Fatal(err)
	}
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("%s=%v", key, m[key])
	}
	return int(v)
}

func TestOnMessageUTFNotReplayedOnLaterPackets(t *testing.T) {
	c := NewController(&fakeInj{})
	c.SetSnapshot(Snapshot{InGame: true})
	msg := protocol.NewMessage(25)
	msg.WriteUTF("Thành công")
	c.OnMessage(msg)
	if got := statusInt(t, c, "success"); got != 1 {
		t.Fatalf("success=%d want 1", got)
	}
	c.OnMessage(protocol.NewMessage(-100))
	if got := statusInt(t, c, "success"); got != 1 {
		t.Fatalf("success replayed on later packet: %d", got)
	}
}

func TestLoopAbortsCycleWhenNotInGame(t *testing.T) {
	inj := &fakeInj{}
	c := NewController(inj)
	items := []*Item{
		{ID: 1, Type: 0, Name: "Kiem", Level: 6, Quantity: 1},
		{ID: 9, Type: 7, Name: "Co 3 la", Quantity: 5, IsClover3: true},
	}
	c.SetSnapshot(Snapshot{InGame: true, Items: items})
	if err := c.Start(Config{
		ItemID: 1, ItemType: 0, TargetLevel: 7,
		Clover3Min: 4, Clover3Max: 7,
		DelayMs: 5, StopIfOutOfClover: true,
	}); err != "" {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		inj.mu.Lock()
		n := len(inj.msgs)
		inj.mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	c.SetSnapshot(Snapshot{InGame: false, Items: items})
	time.Sleep(20 * time.Millisecond)
	inj.mu.Lock()
	nAfterDrop := len(inj.msgs)
	inj.mu.Unlock()
	attemptsAtDrop := statusInt(t, c, "attempts")
	time.Sleep(400 * time.Millisecond)
	c.Stop()
	inj.mu.Lock()
	nFinal := len(inj.msgs)
	inj.mu.Unlock()
	if nFinal > nAfterDrop {
		t.Fatalf("injected %d more packets after !InGame (had %d)", nFinal-nAfterDrop, nAfterDrop)
	}
	if got := statusInt(t, c, "attempts"); got != attemptsAtDrop {
		t.Fatalf("attempts incremented after !InGame: %d -> %d", attemptsAtDrop, got)
	}
}
