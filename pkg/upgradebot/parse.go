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

func IsClover3(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	return strings.Contains(lower, "cỏ 3") || strings.Contains(lower, "co 3") ||
		strings.Contains(lower, "3 lá") || strings.Contains(lower, "3 la") ||
		strings.Contains(lower, "cỏ ba lá") || strings.Contains(lower, "co ba la") ||
		(strings.Contains(lower, "cỏ") && strings.Contains(lower, "3")) ||
		(strings.Contains(lower, "co") && strings.Contains(lower, "3"))
}

func IsClover4(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	return strings.Contains(lower, "cỏ 4") || strings.Contains(lower, "co 4") ||
		strings.Contains(lower, "4 lá") || strings.Contains(lower, "4 la") ||
		strings.Contains(lower, "cỏ bốn lá") || strings.Contains(lower, "co bon la") ||
		(strings.Contains(lower, "cỏ") && strings.Contains(lower, "4")) ||
		(strings.Contains(lower, "co") && strings.Contains(lower, "4"))
}

func IsInsurance(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	return strings.Contains(lower, "bảo hiểm") || strings.Contains(lower, "bao hiem")
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
	if msg.Command == -100 || msg.Command == 24 {
		s.ParseInventory(msg)
	}
}

func (s *Snapshot) ParseInventory(msg *protocol.Message) {
	if msg == nil {
		return
	}
	data := msg.GetData()
	if len(data) == 0 {
		return
	}
	copyMsg := protocol.NewMessageWithData(msg.Command, data)
	gold, err := copyMsg.ReadLong()
	if err == nil {
		s.Gold = gold
	}
	itemCount, err := copyMsg.ReadShort()
	if err != nil || itemCount <= 0 {
		return
	}

	items := make([]*Item, 0, int(itemCount))
	for i := 0; i < int(itemCount); i++ {
		id, err := copyMsg.ReadShort()
		if err != nil {
			break
		}
		name, err := copyMsg.ReadUTF()
		if err != nil {
			break
		}
		itemType, err := copyMsg.ReadInt8()
		if err != nil {
			break
		}
		level, err := copyMsg.ReadInt8()
		if err != nil {
			break
		}
		qty, err := copyMsg.ReadInt()
		if err != nil {
			break
		}
		if qty <= 0 {
			qty = 1
		}

		items = append(items, &Item{
			ID:        id,
			Name:      name,
			Type:      itemType,
			Level:     level,
			Quantity:  int(qty),
			IsClover3: IsClover3(name),
			IsClover4: IsClover4(name),
		})
	}
	if len(items) > 0 {
		s.Items = items
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
