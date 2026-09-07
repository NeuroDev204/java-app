package upgradebot

import "testing"

func TestBuildUpgradeActionLayout(t *testing.T) {
	msg := BuildUpgradeAction(ActionUpgrade, 101, 0)
	if msg.Command != ActionCommand {
		t.Fatalf("cmd=%d", msg.Command)
	}
	a, _ := msg.ReadByte()
	id, _ := msg.ReadShort()
	typ, _ := msg.ReadByte()
	if a != ActionUpgrade || id != 101 || typ != 0 {
		t.Fatalf("payload %d %d %d", a, id, typ)
	}
}
