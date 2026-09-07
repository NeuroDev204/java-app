// Package game defines core domain models and thread-safe inventory state management
// for the knight-auto-upgrade bot.
package game

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Item represents an in-game inventory item, weapon, armor, or upgrade material.
type Item struct {
	ID          int16  `json:"id"`
	Name        string `json:"name"`
	Type        int8   `json:"type"`
	Level       int8   `json:"level"`
	Quantity    int    `json:"quantity"`
	IsClover3   bool   `json:"is_clover_3"`
	IsClover4   bool   `json:"is_clover_4"`
	IsInsurance bool   `json:"is_insurance"`
	IsStone     bool   `json:"is_stone"`
	StoneTier   int8   `json:"stone_tier"`
}

var (
	stoneTierRegex = regexp.MustCompile(`(?i)(?:cấp|cap|tier|lv|level)\s*(\d+)`)
)

// NewItem constructs a new Item and automatically detects its special material properties
// based on name keywords and item type.
func NewItem(id int16, name string, itemType int8, level int8, quantity int) *Item {
	it := &Item{
		ID:       id,
		Name:     name,
		Type:     itemType,
		Level:    level,
		Quantity: quantity,
	}
	it.DetectProperties()
	return it
}

// DetectProperties inspects the item name and type to automatically classify materials
// such as Cỏ 3 lá, Cỏ 4 lá, Bảo hiểm, and Đá cường hóa.
func (it *Item) DetectProperties() {
	normalized := strings.ToLower(it.Name)

	// Clover 4 detection (check before Clover 3 to avoid substring confusion)
	if strings.Contains(normalized, "cỏ 4 lá") ||
		strings.Contains(normalized, "co 4 la") ||
		strings.Contains(normalized, "4 lá") ||
		strings.Contains(normalized, "4 la") ||
		strings.Contains(normalized, "clover 4") ||
		strings.Contains(normalized, "cỏ bốn lá") ||
		strings.Contains(normalized, "co bon la") {
		it.IsClover4 = true
		return
	}

	// Clover 3 detection
	if strings.Contains(normalized, "cỏ 3 lá") ||
		strings.Contains(normalized, "co 3 la") ||
		strings.Contains(normalized, "3 lá") ||
		strings.Contains(normalized, "3 la") ||
		strings.Contains(normalized, "clover 3") ||
		strings.Contains(normalized, "cỏ ba lá") ||
		strings.Contains(normalized, "co ba la") {
		it.IsClover3 = true
		return
	}

	// Insurance detection
	if strings.Contains(normalized, "bảo hiểm") ||
		strings.Contains(normalized, "bao hiem") ||
		strings.Contains(normalized, "insurance") {
		it.IsInsurance = true
		return
	}

	// Upgrade stone detection
	if strings.Contains(normalized, "đá") || strings.Contains(normalized, "da ") || strings.HasPrefix(normalized, "da") {
		if strings.Contains(normalized, "cường hóa") ||
			strings.Contains(normalized, "cuong hoa") ||
			strings.Contains(normalized, "cấp") ||
			strings.Contains(normalized, "cap") ||
			strings.Contains(normalized, "may mắn") ||
			strings.Contains(normalized, "may man") {
			it.IsStone = true
			matches := stoneTierRegex.FindStringSubmatch(normalized)
			if len(matches) > 1 {
				if tier, err := strconv.Atoi(matches[1]); err == nil {
					it.StoneTier = int8(tier)
				}
			}
		}
	}
}

// IsEquipment returns true if the item is an equipable gear (weapon, armor, jewelry)
// rather than a material or consumable.
func (it *Item) IsEquipment() bool {
	if it.IsClover3 || it.IsClover4 || it.IsInsurance || it.IsStone {
		return false
	}
	// In HSO, Type 7 is consumable/material, types 0..6 and other positive non-material types are equipment
	return it.Type != 7
}

// Clone returns an independent deep copy of the Item.
func (it *Item) Clone() *Item {
	if it == nil {
		return nil
	}
	cpy := *it
	return &cpy
}

// String returns a human-readable representation of the item.
func (it *Item) String() string {
	if it.Level > 0 {
		return fmt.Sprintf("%s (+%d) [ID: %d, Qty: %d]", it.Name, it.Level, it.ID, it.Quantity)
	}
	return fmt.Sprintf("%s [ID: %d, Qty: %d]", it.Name, it.ID, it.Quantity)
}
