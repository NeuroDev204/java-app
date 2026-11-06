package gamebridge

import (
	"encoding/json"
	"testing"

	"java-app/pkg/classfile"
	"java-app/pkg/upgradebot"
	"java-app/pkg/vm"
)

func TestGetInventoryJSONFromVM(t *testing.T) {
	b := NewBridge()

	// Create dummy class files
	emptyCF := &classfile.ClassFile{ConstantPool: classfile.ConstantPool{}}
	b.vmCtx = vm.NewVMContext(nil, nil)

	// Set player gold in bq.g
	b.vmCtx.SetStaticField("bq.g", vm.Slot{Num: 99999})

	// Create an et mVector object wrapping VectorData
	etObj := vm.NewObject(emptyCF)
	vecObj := vm.NewObject(emptyCF)
	vd := vm.NewVectorData()
	vecObj.Extra = vd
	etObj.SetField("a", vm.Slot{Ref: vecObj})

	// Item 1: Sword +8 (type 0)
	swordObj := vm.NewObject(emptyCF)
	swordObj.SetField("O", vm.Slot{Num: 555})
	swordObj.SetField("u", vm.Slot{Num: 0})
	swordObj.SetField("z", vm.Slot{Num: 8})
	swordObj.SetField("g", vm.Slot{Ref: b.vmCtx.NewString("Thanh Kiếm Huyền Thoại")})
	swordObj.SetField("p", vm.Slot{Num: 1})
	vd.Add(swordObj)

	// Item 2: Co 3 la (type 7, mat 11)
	c3Obj := vm.NewObject(emptyCF)
	c3Obj.SetField("O", vm.Slot{Num: 201})
	c3Obj.SetField("u", vm.Slot{Num: 7})
	c3Obj.SetField("A", vm.Slot{Num: 11})
	c3Obj.SetField("g", vm.Slot{Ref: b.vmCtx.NewString("Cỏ 3 lá may mắn")})
	c3Obj.SetField("p", vm.Slot{Num: 14})
	vd.Add(c3Obj)

	// Item 3: Co 4 la (type 7, mat 12)
	c4Obj := vm.NewObject(emptyCF)
	c4Obj.SetField("O", vm.Slot{Num: 202})
	c4Obj.SetField("u", vm.Slot{Num: 7})
	c4Obj.SetField("A", vm.Slot{Num: 12})
	c4Obj.SetField("g", vm.Slot{Ref: b.vmCtx.NewString("Cỏ 4 lá đặc biệt")})
	c4Obj.SetField("p", vm.Slot{Num: 7})
	vd.Add(c4Obj)

	// Set static bw.V
	b.vmCtx.SetStaticField("bw.V", vm.Slot{Ref: etObj})

	// Also attach a bot controller to verify sync
	bot := upgradebot.NewController(nil)
	b.bot = bot

	jsonStr := b.GetInventoryJSON()
	t.Logf("Resulting Inventory JSON: %s", jsonStr)

	var parsed struct {
		Gold           int `json:"gold"`
		Clover3Count   int `json:"clover3Count"`
		Clover4Count   int `json:"clover4Count"`
		InsuranceCount int `json:"insuranceCount"`
		Items          []struct {
			ID       int16  `json:"id"`
			Name     string `json:"name"`
			Type     int8   `json:"type"`
			Level    int8   `json:"level"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if parsed.Gold != 99999 {
		t.Errorf("expected gold 99999, got %d", parsed.Gold)
	}
	if parsed.Clover3Count != 14 {
		t.Errorf("expected clover3 14, got %d", parsed.Clover3Count)
	}
	if parsed.Clover4Count != 7 {
		t.Errorf("expected clover4 7, got %d", parsed.Clover4Count)
	}
	if len(parsed.Items) != 1 {
		t.Fatalf("expected 1 equipment item, got %d", len(parsed.Items))
	}
	if parsed.Items[0].Name != "Thanh Kiếm Huyền Thoại" {
		t.Errorf("expected sword name, got %s", parsed.Items[0].Name)
	}
	if parsed.Items[0].Level != 8 {
		t.Errorf("expected level 8, got %d", parsed.Items[0].Level)
	}

	// Verify bot was synchronized
	snap := bot.GetSnapshot()
	if !snap.InGame {
		t.Error("expected bot snapshot to be InGame")
	}
	if snap.Gold != 99999 {
		t.Errorf("expected bot gold 99999, got %d", snap.Gold)
	}
	if snap.Clover3() == nil || snap.Clover3().Quantity != 14 {
		t.Errorf("expected bot clover3 qty 14, got %+v", snap.Clover3())
	}
	if snap.Clover4() == nil || snap.Clover4().Quantity != 7 {
		t.Errorf("expected bot clover4 qty 7, got %+v", snap.Clover4())
	}
}
