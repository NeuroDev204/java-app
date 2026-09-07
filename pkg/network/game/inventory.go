// Package game defines core domain models and thread-safe inventory state management
// for the knight-auto-upgrade bot.
package game

import (
	"sync"
)

// Inventory manages the player's currency (gold) and item collection in a thread-safe manner.
type Inventory struct {
	mu    sync.RWMutex
	gold  int64
	items map[int16]*Item
}

// NewInventory creates an empty, initialized Inventory instance.
func NewInventory() *Inventory {
	return &Inventory{
		items: make(map[int16]*Item),
	}
}

// UpdateGold updates the current gold balance.
func (inv *Inventory) UpdateGold(gold int64) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.gold = gold
}

// GetGold returns the current gold balance.
func (inv *Inventory) GetGold() int64 {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.gold
}

// AddOrUpdateItem adds an item to inventory or updates an existing item.
// If the item quantity is 0 or negative, it is removed from the inventory.
func (inv *Inventory) AddOrUpdateItem(item *Item) {
	if item == nil {
		return
	}

	inv.mu.Lock()
	defer inv.mu.Unlock()

	if item.Quantity <= 0 {
		delete(inv.items, item.ID)
		return
	}

	cloned := item.Clone()
	if !cloned.IsClover3 && !cloned.IsClover4 && !cloned.IsInsurance && !cloned.IsStone {
		cloned.DetectProperties()
	}

	inv.items[cloned.ID] = cloned
}

// RemoveItem decreases the quantity of an item with the given ID.
// If the quantity drops to 0 or below, the item is removed from the inventory.
func (inv *Inventory) RemoveItem(id int16, qty int) {
	if qty <= 0 {
		return
	}

	inv.mu.Lock()
	defer inv.mu.Unlock()

	item, exists := inv.items[id]
	if !exists {
		return
	}

	item.Quantity -= qty
	if item.Quantity <= 0 {
		delete(inv.items, id)
	}
}

// GetItem retrieves a cloned copy of an item by ID, or nil if not present.
func (inv *Inventory) GetItem(id int16) *Item {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	item, exists := inv.items[id]
	if !exists {
		return nil
	}
	return item.Clone()
}

// GetEquipments returns a cloned slice of all equipment items currently in the inventory.
func (inv *Inventory) GetEquipments() []*Item {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	equipments := make([]*Item, 0)
	for _, item := range inv.items {
		if item.IsEquipment() {
			equipments = append(equipments, item.Clone())
		}
	}
	return equipments
}

// GetClover3Count returns the total quantity of Cỏ 3 lá in the inventory.
func (inv *Inventory) GetClover3Count() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	total := 0
	for _, item := range inv.items {
		if item.IsClover3 {
			total += item.Quantity
		}
	}
	return total
}

// GetClover4Count returns the total quantity of Cỏ 4 lá in the inventory.
func (inv *Inventory) GetClover4Count() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	total := 0
	for _, item := range inv.items {
		if item.IsClover4 {
			total += item.Quantity
		}
	}
	return total
}

// GetInsuranceCount returns the total quantity of Bảo hiểm in the inventory.
func (inv *Inventory) GetInsuranceCount() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	total := 0
	for _, item := range inv.items {
		if item.IsInsurance {
			total += item.Quantity
		}
	}
	return total
}

// GetStonesByTier returns the total quantity of enhancement stones of a specific tier.
func (inv *Inventory) GetStonesByTier(tier int8) int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	total := 0
	for _, item := range inv.items {
		if item.IsStone && item.StoneTier == tier {
			total += item.Quantity
		}
	}
	return total
}

// GetAllItems returns a cloned slice of all items in the inventory.
func (inv *Inventory) GetAllItems() []*Item {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	items := make([]*Item, 0, len(inv.items))
	for _, item := range inv.items {
		items = append(items, item.Clone())
	}
	return items
}

// ItemCount returns the number of distinct item entries in the inventory.
func (inv *Inventory) ItemCount() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return len(inv.items)
}

// Clear removes all items and resets the gold balance.
func (inv *Inventory) Clear() {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.items = make(map[int16]*Item)
	inv.gold = 0
}
