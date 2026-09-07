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
