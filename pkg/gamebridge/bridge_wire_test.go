package gamebridge

import (
	"encoding/json"
	"net"
	"testing"

	"java-app/pkg/classfile"
	"java-app/pkg/midp"
	"java-app/pkg/network/protocol"
	"java-app/pkg/vm"
)

const upgradeCfgJSON = `{"item_id":1,"item_type":0,"target_level":10,"clover3_min":4,"clover3_max":7,"clover4_min":8,"clover4_max":15,"stop_if_out_of_clover":true,"delay_ms":500}`

type upgradeStatus struct {
	Running   bool `json:"running"`
	Connected bool `json:"connected"`
	InGame    bool `json:"in_game"`
}

func parseUpgradeStatus(t *testing.T, b *Bridge) upgradeStatus {
	t.Helper()
	var st upgradeStatus
	if err := json.Unmarshal([]byte(b.GetUpgradeStatus()), &st); err != nil {
		t.Fatalf("status json: %v", err)
	}
	return st
}

func newPipeTap(t *testing.T) *midp.SocketTap {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	return midp.NewSocketTap(client)
}

func TestMaybeWireTapDisconnectNotifiesBot(t *testing.T) {
	b := NewBridge()
	tap := newPipeTap(t)
	b.vmCtx = &vm.VMContext{ActiveSocket: tap}
	b.maybeWireTap()
	if b.bot == nil {
		t.Fatal("expected bot after first wire")
	}
	b.bot.OnMessage(protocol.NewMessage(-100))
	st := parseUpgradeStatus(t, b)
	if !st.Connected || !st.InGame {
		t.Fatalf("after in-game opcode: %+v", st)
	}

	b.vmCtx.ActiveSocket = nil
	b.maybeWireTap()
	if b.wiredTap != nil {
		t.Fatal("expected wiredTap nil after disconnect")
	}
	st = parseUpgradeStatus(t, b)
	if st.Connected {
		t.Fatal("expected connected=false after NotifyDisconnected")
	}
	if got := b.StartUpgrade(upgradeCfgJSON); got != "not connected" {
		t.Fatalf("StartUpgrade after drop: %q", got)
	}
}

func TestMaybeWireTapReconnectResetsInGame(t *testing.T) {
	b := NewBridge()
	b.vmCtx = &vm.VMContext{ActiveSocket: newPipeTap(t)}
	b.maybeWireTap()
	b.bot.OnMessage(protocol.NewMessage(24))
	if st := parseUpgradeStatus(t, b); !st.InGame || !st.Connected {
		t.Fatalf("before reconnect: %+v", st)
	}

	b.vmCtx.ActiveSocket = newPipeTap(t)
	b.maybeWireTap()
	st := parseUpgradeStatus(t, b)
	if !st.Connected {
		t.Fatal("expected connected=true on new Controller")
	}
	if st.InGame {
		t.Fatal("expected in_game=false after tap identity change")
	}
	if got := b.StartUpgrade(upgradeCfgJSON); got != "not in game" {
		t.Fatalf("StartUpgrade after reconnect: %q", got)
	}
}

func TestStartUpgradeNoInventory(t *testing.T) {
	b := NewBridge()
	b.vmCtx = &vm.VMContext{ActiveSocket: newPipeTap(t)}
	b.maybeWireTap()
	b.bot.OnMessage(protocol.NewMessage(-100))
	got := b.StartUpgrade(upgradeCfgJSON)
	if got != "no inventory" {
		t.Fatalf("StartUpgrade without items: %q", got)
	}
}

func TestDispatchTouchHitsCurrentCanvas(t *testing.T) {
	b := NewBridge()
	b.vmCtx = vm.NewVMContext(nil, midp.NewGraphics(240, 320))

	midletCF := touchTargetClass("TemMidlet", true)
	canvasCF := touchTargetClass("GameCanvas", false)
	b.midletCF = midletCF
	b.midletObj = vm.NewObject(midletCF)
	canvas := vm.NewObject(canvasCF)
	b.vmCtx.CurrentCanvas = canvas

	b.dispatchTouch(touchEvent{x: 12, y: 34, pressed: true})
	if got := canvas.GetField("lastX").Num; got != 12 {
		t.Fatalf("canvas lastX=%d want 12 (midlet sentinel would be 9999)", got)
	}
	if got := canvas.GetField("lastY").Num; got != 34 {
		t.Fatalf("canvas lastY=%d want 34", got)
	}
	if got := b.midletObj.GetField("lastX").Num; got != 0 {
		t.Fatalf("midlet lastX=%d, pointerPressed must not hit TemMidlet", got)
	}

	b.dispatchTouch(touchEvent{x: 40, y: 50, dragged: true})
	if canvas.GetField("lastX").Num != 40 || canvas.GetField("lastY").Num != 50 {
		t.Fatalf("drag canvas=(%d,%d)", canvas.GetField("lastX").Num, canvas.GetField("lastY").Num)
	}

	b.vmCtx.CurrentCanvas = nil
	b.dispatchTouch(touchEvent{x: 7, y: 8, pressed: true})
	if got := b.midletObj.GetField("lastX").Num; got != 9999 {
		t.Fatalf("midlet fallback lastX=%d want 9999", got)
	}
}

func touchTargetClass(name string, sentinel bool) *classfile.ClassFile {
	const (
		idxName = 1
		idxCls  = 2
		idxLastX = 3
		idxI     = 4
		idxNTx   = 5
		idxFRx   = 6
		idxLastY = 7
		idxNTy   = 8
		idxFRy   = 9
		idxPressed = 10
		idxDesc    = 11
		idxDragged = 12
		idxReleased = 13
	)
	storeXY := func() []byte {
		if sentinel {
			return []byte{
				0x2A,             // aload_0
				0x11, 0x27, 0x0F, // sipush 9999
				0xB5, 0x00, idxFRx,
				0x2A,
				0x11, 0x27, 0x0F,
				0xB5, 0x00, idxFRy,
				0xB1,
			}
		}
		return []byte{
			0x2A, 0x1B, 0xB5, 0x00, idxFRx,
			0x2A, 0x1C, 0xB5, 0x00, idxFRy,
			0xB1,
		}
	}
	code := storeXY()
	mk := func(nameIdx uint16) classfile.MethodInfo {
		return classfile.MethodInfo{
			NameIndex: nameIdx,
			DescIndex: idxDesc,
			Code:      append([]byte(nil), code...),
			MaxStack:  2,
			MaxLocals: 3,
		}
	}
	return &classfile.ClassFile{
		ConstantPool: classfile.ConstantPool{
			nil,
			classfile.ConstantUtf8Info{Str: name},
			classfile.ConstantClassInfo{NameIndex: idxName},
			classfile.ConstantUtf8Info{Str: "lastX"},
			classfile.ConstantUtf8Info{Str: "I"},
			classfile.ConstantNameAndTypeInfo{NameIndex: idxLastX, DescriptorIndex: idxI},
			classfile.ConstantMemberRefInfo{ClassIndex: idxCls, NameAndTypeIndex: idxNTx},
			classfile.ConstantUtf8Info{Str: "lastY"},
			classfile.ConstantNameAndTypeInfo{NameIndex: idxLastY, DescriptorIndex: idxI},
			classfile.ConstantMemberRefInfo{ClassIndex: idxCls, NameAndTypeIndex: idxNTy},
			classfile.ConstantUtf8Info{Str: "pointerPressed"},
			classfile.ConstantUtf8Info{Str: "(II)V"},
			classfile.ConstantUtf8Info{Str: "pointerDragged"},
			classfile.ConstantUtf8Info{Str: "pointerReleased"},
		},
		ThisClass: idxCls,
		Methods: []classfile.MethodInfo{
			mk(idxPressed),
			mk(idxDragged),
			mk(idxReleased),
		},
	}
}

func TestDispatchKeyHitsCurrentCanvas(t *testing.T) {
	b := NewBridge()
	b.vmCtx = vm.NewVMContext(nil, midp.NewGraphics(240, 320))

	canvasCF := keyTargetClass("GameCanvas")
	canvas := vm.NewObject(canvasCF)
	b.vmCtx.CurrentCanvas = canvas

	b.SendKeyDown(-5)
	if got := canvas.GetField("lastKey").Num; got != -5 {
		t.Fatalf("canvas lastKey=%d want -5", got)
	}
	if got := canvas.GetField("isPressed").Num; got != 1 {
		t.Fatalf("canvas isPressed=%d want 1", got)
	}

	b.SendKeyUp(-5)
	if got := canvas.GetField("lastKey").Num; got != -5 {
		t.Fatalf("canvas lastKey=%d want -5", got)
	}
	if got := canvas.GetField("isPressed").Num; got != 0 {
		t.Fatalf("canvas isPressed=%d want 0", got)
	}
}

func keyTargetClass(name string) *classfile.ClassFile {
	const (
		idxName      = 1
		idxCls       = 2
		idxLastKey   = 3
		idxI         = 4
		idxNTKey     = 5
		idxFRKey     = 6
		idxIsPressed = 7
		idxNTPressed = 8
		idxFRPressed = 9
		idxPressed   = 10
		idxDesc      = 11
		idxReleased  = 12
	)
	storeKey := func(pressed bool) []byte {
		val := byte(0x03) // iconst_0
		if pressed {
			val = 0x04 // iconst_1
		}
		return []byte{
			0x2A,             // aload_0
			0x1B,             // iload_1
			0xB5, 0x00, idxFRKey, // putfield lastKey
			0x2A,             // aload_0
			val,              // iconst_0 / iconst_1
			0xB5, 0x00, idxFRPressed, // putfield isPressed
			0xB1,             // return
		}
	}
	mk := func(nameIdx uint16, pressed bool) classfile.MethodInfo {
		code := storeKey(pressed)
		return classfile.MethodInfo{
			NameIndex: nameIdx,
			DescIndex: idxDesc,
			Code:      code,
			MaxStack:  2,
			MaxLocals: 2,
		}
	}
	return &classfile.ClassFile{
		ConstantPool: classfile.ConstantPool{
			nil,
			classfile.ConstantUtf8Info{Str: name},
			classfile.ConstantClassInfo{NameIndex: idxName},
			classfile.ConstantUtf8Info{Str: "lastKey"},
			classfile.ConstantUtf8Info{Str: "I"},
			classfile.ConstantNameAndTypeInfo{NameIndex: idxLastKey, DescriptorIndex: idxI},
			classfile.ConstantMemberRefInfo{ClassIndex: idxCls, NameAndTypeIndex: idxNTKey},
			classfile.ConstantUtf8Info{Str: "isPressed"},
			classfile.ConstantNameAndTypeInfo{NameIndex: idxIsPressed, DescriptorIndex: idxI},
			classfile.ConstantMemberRefInfo{ClassIndex: idxCls, NameAndTypeIndex: idxNTPressed},
			classfile.ConstantUtf8Info{Str: "keyPressed"},
			classfile.ConstantUtf8Info{Str: "(I)V"},
			classfile.ConstantUtf8Info{Str: "keyReleased"},
		},
		ThisClass: idxCls,
		Methods: []classfile.MethodInfo{
			mk(idxPressed, true),
			mk(idxReleased, false),
		},
	}
}
