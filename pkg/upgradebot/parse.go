package upgradebot

import (
	"strings"

	"java-app/pkg/network/protocol"
)

func IsInGameOpcode(cmd int8) bool {
	return cmd == -100 || cmd == 24 || cmd == 26
}

func IsStopCondition(msg string) bool {
	if msg == "" {
		return false
	}
	lower := strings.ToLower(msg)
	if IsIgnorableResult(msg) {
		return false
	}
	needles := []string{
		"không đủ xu", "khong du xu",
		"không đủ vàng", "khong du vang",
		"không đủ tiền", "khong du tien",
		"không đủ ngọc", "khong du ngoc",
		"không đủ đá", "khong du da",
		"thiếu đá", "thieu da",
		"hành trang đầy", "hanh trang day",
		"hành trang không đủ", "hanh trang khong du",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

func IsIgnorableResult(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "cố gắng hết sức") ||
		strings.Contains(lower, "co gang het suc") ||
		strings.Contains(lower, "thành công") ||
		strings.Contains(lower, "thanh cong") ||
		strings.Contains(lower, "thất bại") ||
		strings.Contains(lower, "that bai")
}

type Item struct {
	ID        int16  `json:"id"`
	Name      string `json:"name"`
	Type      int8   `json:"type"`
	Level     int8   `json:"level"`
	Quantity  int    `json:"quantity"`
	IsClover3 bool   `json:"is_clover_3"`
	IsClover4 bool   `json:"is_clover_4"`
}

type Snapshot struct {
	InGame        bool
	Gold          int64
	Items         []*Item
	LastServerMsg string
}

func (s *Snapshot) Apply(msg *protocol.Message) {
	if msg == nil {
		return
	}
	if IsInGameOpcode(msg.Command) {
		s.InGame = true
	}
}

func (s *Snapshot) Clover3() *Item {
	for _, it := range s.Items {
		if it.IsClover3 && it.Quantity > 0 {
			return it
		}
	}
	return nil
}

func (s *Snapshot) Clover4() *Item {
	for _, it := range s.Items {
		if it.IsClover4 && it.Quantity > 0 {
			return it
		}
	}
	return nil
}

func (s *Snapshot) Find(id int16, typ int8) *Item {
	for _, it := range s.Items {
		if it.ID == id && it.Type == typ {
			return it
		}
	}
	return nil
}
