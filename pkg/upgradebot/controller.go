package upgradebot

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"java-app/pkg/network/protocol"
)

type Injector interface {
	Inject(msg *protocol.Message) error
}

type Controller struct {
	inj Injector

	mu        sync.Mutex
	cfg       Config
	snap      Snapshot
	running   bool
	connected bool
	attempts  int
	success   int
	fail      int
	logs      []string
	gen       int
}

func NewController(inj Injector) *Controller {
	return &Controller{
		inj:       inj,
		connected: true,
		logs:      []string{},
	}
}

func (c *Controller) Start(cfg Config) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.snap.InGame {
		return "not in game"
	}
	item := c.snap.Find(cfg.ItemID, cfg.ItemType)
	if item == nil {
		return "no inventory"
	}
	if d := Evaluate(cfg, item.Level, c.snap); !d.CanProceed {
		if d.Reason != "" {
			return d.Reason
		}
		return "no inventory"
	}
	c.cfg = cfg
	c.snap.LastServerMsg = ""
	c.running = true
	c.gen++
	gen := c.gen
	c.appendLogLocked(fmt.Sprintf("Bắt đầu Auto-Upgrade +%d", cfg.TargetLevel))
	go c.loop(gen)
	return ""
}

func (c *Controller) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
}

func (c *Controller) stopLocked() {
	if !c.running {
		return
	}
	c.running = false
	c.appendLogLocked("Đã dừng Auto-Upgrade.")
}

func (c *Controller) Running() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *Controller) SetSnapshot(s Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snap = s
}

func (c *Controller) NotifyDisconnected() {
	c.mu.Lock()
	c.connected = false
	c.stopLocked()
	c.mu.Unlock()
}

func (c *Controller) OnMessage(msg *protocol.Message) {
	if msg == nil {
		return
	}

	c.mu.Lock()
	c.snap.Apply(msg)
	utf := ""
	gotUTF := false
	if msg.Command == 25 || msg.Command == 37 {
		if s, err := msg.ReadUTF(); err == nil {
			utf = s
			gotUTF = true
			c.snap.LastServerMsg = s
		}
	}
	cfg := c.cfg
	running := c.running
	item := c.snap.Find(cfg.ItemID, cfg.ItemType)
	var level int8
	if item != nil {
		level = item.Level
	}
	c.mu.Unlock()

	if gotUTF {
		if IsStopCondition(utf) {
			c.Stop()
			return
		}
		if IsIgnorableResult(utf) {
			lower := strings.ToLower(utf)
			c.mu.Lock()
			if strings.Contains(lower, "thành công") || strings.Contains(lower, "thanh cong") {
				c.success++
				c.appendLogLocked(utf)
			} else if strings.Contains(lower, "thất bại") || strings.Contains(lower, "that bai") ||
				strings.Contains(lower, "cố gắng hết sức") || strings.Contains(lower, "co gang het suc") {
				c.fail++
				c.appendLogLocked(utf)
			}
			c.mu.Unlock()
		}
	}
	if running && item != nil && level >= cfg.TargetLevel {
		c.mu.Lock()
		c.success++
		c.mu.Unlock()
		c.Stop()
	}
}

func (c *Controller) StatusJSON() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var itemLevel int8
	if it := c.snap.Find(c.cfg.ItemID, c.cfg.ItemType); it != nil {
		itemLevel = it.Level
	}
	logs := append([]string(nil), c.logs...)
	raw, err := json.Marshal(struct {
		Running   bool     `json:"running"`
		Connected bool     `json:"connected"`
		InGame    bool     `json:"in_game"`
		Attempts  int      `json:"attempts"`
		Success   int      `json:"success"`
		Fail      int      `json:"fail"`
		ItemLevel int8     `json:"item_level"`
		Logs      []string `json:"logs"`
	}{
		Running:   c.running,
		Connected: c.connected,
		InGame:    c.snap.InGame,
		Attempts:  c.attempts,
		Success:   c.success,
		Fail:      c.fail,
		ItemLevel: itemLevel,
		Logs:      logs,
	})
	if err != nil {
		return `{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}`
	}
	return string(raw)
}

func (c *Controller) loop(gen int) {
	for {
		c.mu.Lock()
		if !c.running || c.gen != gen {
			c.mu.Unlock()
			return
		}
		inGame := c.snap.InGame
		cfg := c.cfg
		snap := c.snap
		delay := stepDelay(cfg.DelayMs)
		c.mu.Unlock()

		if !inGame {
			if !c.sleepStep(delay, gen) {
				return
			}
			continue
		}

		var level int8
		if it := snap.Find(cfg.ItemID, cfg.ItemType); it != nil {
			level = it.Level
		}
		d := Evaluate(cfg, level, snap)
		if !d.CanProceed {
			c.Stop()
			return
		}

		if !c.injectAction(gen, ActionSelect, cfg.ItemID, cfg.ItemType) {
			if !c.abortCycleOrStop(gen, delay) {
				return
			}
			continue
		}
		if !c.sleepStep(delay, gen) {
			return
		}

		if d.UseClover3 {
			if cl := snap.Clover3(); cl != nil {
				if !c.injectAction(gen, ActionSelect, cl.ID, cl.Type) {
					if !c.abortCycleOrStop(gen, delay) {
						return
					}
					continue
				}
				if !c.sleepStep(delay, gen) {
					return
				}
			}
		}
		if d.UseClover4 {
			if cl := snap.Clover4(); cl != nil {
				if !c.injectAction(gen, ActionSelect, cl.ID, cl.Type) {
					if !c.abortCycleOrStop(gen, delay) {
						return
					}
					continue
				}
				if !c.sleepStep(delay, gen) {
					return
				}
			}
		}

		if !c.injectAction(gen, ActionCurrency, 0, 0) {
			if !c.abortCycleOrStop(gen, delay) {
				return
			}
			continue
		}
		if !c.sleepStep(delay, gen) {
			return
		}

		if !c.injectAction(gen, ActionUpgrade, cfg.ItemID, cfg.ItemType) {
			if !c.abortCycleOrStop(gen, delay) {
				return
			}
			continue
		}

		c.mu.Lock()
		if c.running && c.gen == gen {
			c.attempts++
		}
		c.mu.Unlock()

		if !c.sleepStep(delay, gen) {
			return
		}
	}
}

func (c *Controller) injectAction(gen int, action byte, itemID int16, itemType int8) bool {
	c.mu.Lock()
	if !c.running || c.gen != gen || !c.snap.InGame {
		c.mu.Unlock()
		return false
	}
	inj := c.inj
	c.mu.Unlock()
	if inj == nil {
		c.Stop()
		return false
	}
	if err := inj.Inject(BuildUpgradeAction(action, itemID, itemType)); err != nil {
		c.Stop()
		return false
	}
	return true
}

// abortCycleOrStop returns true when the loop should skip the rest of this
// cycle (still running, typically !InGame) and false when the goroutine must exit.
func (c *Controller) abortCycleOrStop(gen int, delay time.Duration) bool {
	if !c.stillCurrent(gen) {
		return false
	}
	return c.sleepStep(delay, gen)
}

func (c *Controller) sleepStep(d time.Duration, gen int) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if !c.stillCurrent(gen) {
			return false
		}
		remaining := time.Until(deadline)
		if remaining > 10*time.Millisecond {
			remaining = 10 * time.Millisecond
		}
		time.Sleep(remaining)
	}
	return c.stillCurrent(gen)
}

func (c *Controller) stillCurrent(gen int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running && c.gen == gen
}

func (c *Controller) appendLogLocked(msg string) {
	line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	c.logs = append([]string{line}, c.logs...)
	if len(c.logs) > 100 {
		c.logs = c.logs[:100]
	}
}

func stepDelay(ms int) time.Duration {
	if ms < 120 {
		ms = 120
	}
	return time.Duration(ms) * time.Millisecond
}

// InventoryJSON returns JSON representing the current bag items from the snapshot.
func (c *Controller) InventoryJSON() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	c3 := 0
	if it := c.snap.Clover3(); it != nil {
		c3 = it.Quantity
	}
	c4 := 0
	if it := c.snap.Clover4(); it != nil {
		c4 = it.Quantity
	}

	type invItem struct {
		ID       int16  `json:"id"`
		Name     string `json:"name"`
		Type     int8   `json:"type"`
		Level    int8   `json:"level"`
		Quantity int    `json:"quantity"`
	}
	items := make([]invItem, 0, len(c.snap.Items))
	for _, it := range c.snap.Items {
		if it != nil {
			items = append(items, invItem{
				ID:       it.ID,
				Name:     it.Name,
				Type:     it.Type,
				Level:    it.Level,
				Quantity: it.Quantity,
			})
		}
	}

	raw, err := json.Marshal(map[string]interface{}{
		"gold":           c.snap.Gold,
		"clover3Count":   c3,
		"clover4Count":   c4,
		"insuranceCount": 0,
		"items":          items,
	})
	if err != nil {
		return `{"gold":0,"clover3Count":0,"clover4Count":0,"insuranceCount":0,"items":[]}`
	}
	return string(raw)
}

// SyncSnapshot updates the internal snapshot with authoritative state (e.g. from VM memory).
func (c *Controller) SyncSnapshot(inGame bool, gold int64, items []*Item) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snap.InGame = inGame
	if gold > 0 || c.snap.Gold == 0 {
		c.snap.Gold = gold
	}
	if len(items) > 0 {
		c.snap.Items = items
	}
}

// GetSnapshot returns a copy of the current internal snapshot.
func (c *Controller) GetSnapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snap
}

