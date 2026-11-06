package upgradebot

import (
	"testing"

	"java-app/pkg/network/protocol"
)

func TestIsInGameOpcode(t *testing.T) {
	if !IsInGameOpcode(-100) || !IsInGameOpcode(24) || !IsInGameOpcode(26) {
		t.Fatal("expected -100, 24, 26 to mark in_game")
	}
	if IsInGameOpcode(1) {
		t.Fatal("login opcode is not in_game")
	}
}

func TestSnapshotApplySetsInGame(t *testing.T) {
	var s Snapshot
	s.Apply(protocol.NewMessage(-100))
	if !s.InGame {
		t.Fatal("expected InGame")
	}
}

func TestSnapshotParseInventory(t *testing.T) {
	msg := protocol.NewMessage(-100)
	msg.WriteLong(150000) // 150,000 gold
	msg.WriteShort(2)      // 2 items

	// Item 1: Sword
	msg.WriteShort(101)   // id
	msg.WriteUTF("Kiem Sat +7") // name
	msg.WriteInt8(0)      // type
	msg.WriteInt8(7)      // level
	msg.WriteInt(1)       // quantity

	// Item 2: Clover 3
	msg.WriteShort(201)
	msg.WriteUTF("Cỏ 3 lá")
	msg.WriteInt8(7)
	msg.WriteInt8(0)
	msg.WriteInt(15)

	var s Snapshot
	s.Apply(msg)

	if !s.InGame {
		t.Fatal("expected InGame")
	}
	if s.Gold != 150000 {
		t.Fatalf("expected 150000 gold, got %d", s.Gold)
	}
	if len(s.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(s.Items))
	}
	if s.Clover3() == nil || s.Clover3().Quantity != 15 {
		t.Fatalf("expected clover3 with qty 15, got %+v", s.Clover3())
	}
	sword := s.Find(101, 0)
	if sword == nil || sword.Level != 7 {
		t.Fatalf("expected sword at level 7, got %+v", sword)
	}
}

func TestStopConditionVietnamese(t *testing.T) {
	stops := []string{
		"Không đủ xu", "khong du vang", "không đủ ngọc",
		"thiếu đá", "Hành trang đầy", "hanh trang khong du",
	}
	for _, m := range stops {
		if !IsStopCondition(m) {
			t.Fatalf("expected stop: %q", m)
		}
	}
	if IsStopCondition("Cố gắng hết sức") || IsStopCondition("Thành công") || IsStopCondition("Thất bại") {
		t.Fatal("success/fail dialogue must not stop")
	}
}
