package upgradebot

import "java-app/pkg/network/protocol"

const (
	ActionSelect   byte = 0
	ActionCurrency byte = 2
	ActionUpgrade  byte = 5
)

// ActionCommand is the HSO packet command written by KnightOnline_402.jar
// method q.b(byte,short,byte) (opcode 67 / 0x43).
var ActionCommand int8 = 67

func BuildUpgradeAction(action byte, itemID int16, itemType int8) *protocol.Message {
	msg := protocol.NewMessage(ActionCommand)
	_ = msg.WriteByte(action)
	msg.WriteShort(itemID)
	_ = msg.WriteByte(byte(itemType))
	return msg
}
