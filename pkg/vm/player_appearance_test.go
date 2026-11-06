package vm

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestSyncPlayerAppearance(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// Create cn.g player object with default 0 appearance
	cnG := NewObject(nil)
	cnG.SetField("cu", Slot{Num: 12345})
	cnG.SetField("cC", Slot{Ref: "vjncible"})
	cnG.SetField("bZ", Slot{Num: 0})
	cnG.SetField("ca", Slot{Num: 0})
	cnG.SetField("bX", Slot{Num: 0})
	cnG.SetField("cd", Slot{Num: 0})
	cnG.SetField("dY", Slot{Num: 0})
	cnG.SetField("cc", Slot{Num: 0})
	vmCtx.SetStaticField("cn.g", Slot{Ref: cnG})

	// Create cn.j (entity vector)
	cnJVec := NewVectorData()
	cnJObj := NewObject(nil)
	cnJInner := NewObject(nil)
	cnJInner.Extra = cnJVec
	cnJObj.SetField("a", Slot{Ref: cnJInner})
	vmCtx.SetStaticField("cn.j", Slot{Ref: cnJObj})

	// Create x.a (character select vector)
	xAVec := NewVectorData()
	xAObj := NewObject(nil)
	xAInner := NewObject(nil)
	xAInner.Extra = xAVec
	xAObj.SetField("a", Slot{Ref: xAInner})
	vmCtx.SetStaticField("x.a", Slot{Ref: xAObj})

	// Add vjncible character with witch hat, fairy wings, dress, and wand
	bmChar := NewObject(nil)
	bmChar.SetField("cu", Slot{Num: 12345})
	bmChar.SetField("cC", Slot{Ref: "vjncible"})
	bmChar.SetField("bZ", Slot{Num: 15})  // red dress
	bmChar.SetField("ca", Slot{Num: 10})  // legs
	bmChar.SetField("bX", Slot{Num: 12})  // witch hat
	bmChar.SetField("cd", Slot{Num: 21})  // select-screen wing id, not a b*_wing sprite
	bmChar.SetField("dY", Slot{Num: 18})  // select-screen weapon id, out of phapsu 0..17
	bmChar.SetField("cc", Slot{Num: 4})   // blonde hair
	bmChar.SetField("by", Slot{Num: 2})   // phapsu class
	bmChar.SetField("bz", Slot{Num: 101}) // level 101
	xAVec.Add(bmChar)

	// Execute synchronization
	vmCtx.SyncPlayerAppearance()

	// Assertions on cn.g
	if got := cnG.GetField("bZ").Num; got != 15 {
		t.Errorf("expected bZ (body) = 15, got %d", got)
	}
	if got := cnG.GetField("ca").Num; got != 10 {
		t.Errorf("expected ca (legs) = 10, got %d", got)
	}
	if got := cnG.GetField("bX").Num; got != 12 {
		t.Errorf("expected bX (hat) = 12, got %d", got)
	}
	// Wings and weapon must NOT be carried over from character select: those ids
	// address files the JAR does not have, so the layers would draw nothing.
	if got := cnG.GetField("cd").Num; got != 0 {
		t.Errorf("expected cd (wings) left alone = 0, got %d", got)
	}
	if got := cnG.GetField("dY").Num; got != 0 {
		t.Errorf("expected dY (weapon) left alone = 0, got %d", got)
	}
	if got := cnG.GetField("cc").Num; got != 4 {
		t.Errorf("expected cc (hair) = 4, got %d", got)
	}
	if got := cnG.GetField("by").Num; got != 2 {
		t.Errorf("expected by (class) = 2, got %d", got)
	}
	if got := cnG.GetField("bz").Num; got != 101 {
		t.Errorf("expected bz (level) = 101, got %d", got)
	}

	// Assert cn.g is now in cn.j
	if cnJVec.Size() != 1 || cnJVec.Get(0) != cnG {
		t.Errorf("expected cn.g to be in cn.j entity vector, got size=%d", cnJVec.Size())
	}
}

func TestSyncPlayerAppearanceUsesSelectionIndexForFreshPlayer(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// A player created by bq's JAR constructor starts with these defaults. The
	// packet has already filled the identity and the three non-equipment fields.
	cnG := NewObject(nil)
	cnG.SetField("cu", Slot{Num: 0})
	cnG.SetField("cC", Slot{Ref: ""})
	cnG.SetField("bX", Slot{Num: 2})
	cnG.SetField("bY", Slot{Num: 1})
	cnG.SetField("bZ", Slot{Num: 2})
	cnG.SetField("ca", Slot{Num: 2})
	cnG.SetField("cb", Slot{Num: 11})
	cnG.SetField("cc", Slot{Num: 12})
	cnG.SetField("cd", Slot{Num: -1})
	cnG.SetField("ce", Slot{Num: -1})
	vmCtx.SetStaticField("cn.g", Slot{Ref: cnG})

	selectionVector := NewVectorData()
	selection := NewObject(nil)
	selection.SetField("a", Slot{Ref: &Object{Extra: selectionVector}})
	vmCtx.SetStaticField("x.a", Slot{Ref: selection})
	vmCtx.SetStaticField("x.b", Slot{Num: 1})

	firstCharacter := NewObject(nil)
	secondCharacter := NewObject(nil)
	secondCharacter.SetField("bX", Slot{Num: 12})
	secondCharacter.SetField("bZ", Slot{Num: 15})
	secondCharacter.SetField("ca", Slot{Num: 10})
	secondCharacter.SetField("bY", Slot{Num: 7})
	secondCharacter.SetField("cb", Slot{Num: 8})
	secondCharacter.SetField("cc", Slot{Num: 9})
	secondCharacter.SetField("cd", Slot{Num: 21})
	secondCharacter.SetField("ce", Slot{Num: 4})
	selectionVector.Add(firstCharacter)
	selectionVector.Add(secondCharacter)

	vmCtx.SyncPlayerAppearance()

	if got := cnG.GetField("bX").Num; got != 12 {
		t.Errorf("expected selected hat bX=12, got %d", got)
	}
	if got := cnG.GetField("bZ").Num; got != 15 {
		t.Errorf("expected selected body bZ=15, got %d", got)
	}
	if got := cnG.GetField("ca").Num; got != 10 {
		t.Errorf("expected selected legs ca=10, got %d", got)
	}
	if got := cnG.GetField("cd").Num; got != -1 {
		t.Errorf("expected wings cd left alone at -1, got %d", got)
	}
	if got := cnG.GetField("bY").Num; got != 7 {
		t.Errorf("expected selected head bY=7, got %d", got)
	}
	if got := cnG.GetField("cb").Num; got != 8 {
		t.Errorf("expected selected eyes cb=8, got %d", got)
	}
	if got := cnG.GetField("cc").Num; got != 9 {
		t.Errorf("expected selected hair cc=9, got %d", got)
	}
}

func TestSyncPlayerAppearanceUsesMatchingRenderedEntity(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	stalePlayer := NewObject(nil)
	stalePlayer.SetField("cu", Slot{Num: -31061})
	stalePlayer.SetField("cC", Slot{Ref: "vjncible"})
	stalePlayer.SetField("bZ", Slot{Num: 2})
	stalePlayer.SetField("ca", Slot{Num: 2})
	stalePlayer.SetField("bX", Slot{Num: 2})
	stalePlayer.SetField("cd", Slot{Num: -1})
	stalePlayer.SetField("ce", Slot{Num: -1})
	vmCtx.SetStaticField("cn.g", Slot{Ref: stalePlayer})

	renderedPlayer := NewObject(nil)
	renderedPlayer.SetField("cu", Slot{Num: -31061})
	renderedPlayer.SetField("cC", Slot{Ref: "vjncible"})
	renderedPlayer.SetField("bZ", Slot{Num: 2})
	renderedPlayer.SetField("ca", Slot{Num: 2})
	renderedPlayer.SetField("bX", Slot{Num: 2})
	renderedPlayer.SetField("cd", Slot{Num: -1})
	renderedPlayer.SetField("ce", Slot{Num: -1})
	renderedPlayer.SetField("aZ", Slot{Num: 117})
	renderedPlayer.SetField("ba", Slot{Num: 140})
	mapEntities := NewVectorData()
	mapEntities.Add(renderedPlayer)
	mapContainer := NewObject(nil)
	mapContainer.SetField("a", Slot{Ref: &Object{Extra: mapEntities}})
	vmCtx.SetStaticField("cn.j", Slot{Ref: mapContainer})

	selectedPlayer := NewObject(nil)
	selectedPlayer.SetField("cu", Slot{Num: -31061})
	selectedPlayer.SetField("cC", Slot{Ref: "vjncible"})
	selectedPlayer.SetField("bZ", Slot{Num: 39})
	selectedPlayer.SetField("ca", Slot{Num: 39})
	selectedPlayer.SetField("bX", Slot{Num: 39})
	selectedPlayer.SetField("cd", Slot{Num: 21})
	selectedPlayer.SetField("ce", Slot{Num: -1})
	selection := NewVectorData()
	selection.Add(selectedPlayer)
	selectionContainer := NewObject(nil)
	selectionContainer.SetField("a", Slot{Ref: &Object{Extra: selection}})
	vmCtx.SetStaticField("x.a", Slot{Ref: selectionContainer})

	vmCtx.SyncPlayerAppearance()

	if got := vmCtx.GetStaticField("cn.g").Ref; got != renderedPlayer {
		t.Fatal("cn.g still points at a stale object instead of the entity rendered by cn.j")
	}
	if got := renderedPlayer.GetField("bZ").Num; got != 39 {
		t.Fatalf("rendered player body = %d, want 39", got)
	}
	if mapEntities.Size() != 1 {
		t.Fatalf("synchronization inserted a duplicate local player; entity count = %d", mapEntities.Size())
	}
}

func TestImage4000And4001AreValidDistinctRMS(t *testing.T) {
	stores := getBaselineRMSStores()
	store4000, ok4000 := stores["image4000"]
	store4001, ok4001 := stores["image4001"]

	if !ok4000 || store4000 == nil || len(store4000.Records) == 0 {
		t.Fatal("image4000 not found or empty in RecordStores")
	}
	if !ok4001 || store4001 == nil || len(store4001.Records) == 0 {
		t.Fatal("image4001 not found or empty in RecordStores")
	}

	// Authentic image4000 and image4001 must be distinct items (not cloned)
	if bytes.Equal(store4000.Records[0], store4001.Records[0]) {
		t.Fatalf("image4000 should NOT be identical to image4001! Both should be distinct authentic items.")
	}

	// Verify both decrypt to valid PNGs
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for _, store := range []*RecordStoreData{store4000, store4001} {
		data := store.Records[0]
		dec := make([]byte, len(data))
		for i, b := range data {
			dec[i] = b ^ 0xFF
		}
		if !bytes.HasPrefix(dec, pngMagic) {
			t.Fatalf("decrypted record does not have PNG header: % X", dec[:8])
		}
	}
}

func TestSyncPlayerAppearanceNoOverwriteWhenDressed(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// Local player already has server appearance (e.g. from packet -44)
	cnG := NewObject(nil)
	cnG.SetField("cu", Slot{Num: 9999})
	cnG.SetField("cC", Slot{Ref: "mychar"})
	cnG.SetField("bZ", Slot{Num: 5})
	cnG.SetField("ca", Slot{Num: 6})
	cnG.SetField("bX", Slot{Num: 7})
	cnG.SetField("cd", Slot{Num: 0}) // No wings
	cnG.SetField("dY", Slot{Num: 0}) // No weapon
	vmCtx.SetStaticField("cn.g", Slot{Ref: cnG})

	// Vector has character select with wings & weapon
	xAVec := NewVectorData()
	xAObj := NewObject(nil)
	xAInner := NewObject(nil)
	xAInner.Extra = xAVec
	xAObj.SetField("a", Slot{Ref: xAInner})
	vmCtx.SetStaticField("x.a", Slot{Ref: xAObj})

	bmChar := NewObject(nil)
	bmChar.SetField("cu", Slot{Num: 9999})
	bmChar.SetField("cC", Slot{Ref: "mychar"})
	bmChar.SetField("bZ", Slot{Num: 15})
	bmChar.SetField("cd", Slot{Num: 21})
	bmChar.SetField("dY", Slot{Num: 18})
	xAVec.Add(bmChar)

	// SyncPlayerAppearance must NOT overwrite dressed character or force wings/weapons
	vmCtx.SyncPlayerAppearance()

	if got := cnG.GetField("bZ").Num; got != 5 {
		t.Errorf("expected bZ=5 (unchanged), got %d", got)
	}
	if got := cnG.GetField("cd").Num; got != 0 {
		t.Errorf("expected cd=0 (no wings forced), got %d", got)
	}
	if got := cnG.GetField("dY").Num; got != 0 {
		t.Errorf("expected dY=0 (no weapon forced), got %d", got)
	}
}

func TestTransformImageRegion(t *testing.T) {
	// Create a 2x3 test image
	// Row 0: Red, Green
	// Row 1: Blue, Yellow
	// Row 2: White, Black
	src := image.NewRGBA(image.Rect(0, 0, 2, 3))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{G: 255, A: 255})
	src.Set(0, 1, color.RGBA{B: 255, A: 255})
	src.Set(1, 1, color.RGBA{R: 255, G: 255, A: 255})
	src.Set(0, 2, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	src.Set(1, 2, color.RGBA{A: 255})

	// TRANS_NONE (0): 2x3
	res0 := transformImageRegion(src, 0, 0, 2, 3, 0)
	if res0.Bounds().Dx() != 2 || res0.Bounds().Dy() != 3 {
		t.Fatalf("TRANS_NONE expected 2x3, got %dx%d", res0.Bounds().Dx(), res0.Bounds().Dy())
	}
	r, g, b, _ := res0.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("expected Red at (0,0), got r=%d, g=%d, b=%d", r>>8, g>>8, b>>8)
	}

	// TRANS_MIRROR (2 - horizontal flip): (0,0) becomes Green, (1,0) becomes Red
	res2 := transformImageRegion(src, 0, 0, 2, 3, 2)
	if res2.Bounds().Dx() != 2 || res2.Bounds().Dy() != 3 {
		t.Fatalf("TRANS_MIRROR expected 2x3, got %dx%d", res2.Bounds().Dx(), res2.Bounds().Dy())
	}
	r, g, b, _ = res2.At(0, 0).RGBA()
	if r>>8 != 0 || g>>8 != 255 || b>>8 != 0 {
		t.Errorf("expected Green at (0,0) after horizontal flip, got r=%d, g=%d, b=%d", r>>8, g>>8, b>>8)
	}

	// TRANS_ROT90 (5): 2x3 becomes 3x2 (swapped width/height)
	res5 := transformImageRegion(src, 0, 0, 2, 3, 5)
	if res5.Bounds().Dx() != 3 || res5.Bounds().Dy() != 2 {
		t.Fatalf("TRANS_ROT90 expected 3x2, got %dx%d", res5.Bounds().Dx(), res5.Bounds().Dy())
	}
}

func TestCreateRGBImageHonorsProcessAlpha(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	opaqueFrame := &Frame{OperandStack: NewOperandStack(8)}
	opaqueFrame.OperandStack.PushRef([]int32{0x00112233})
	opaqueFrame.OperandStack.PushInt(1)
	opaqueFrame.OperandStack.PushInt(1)
	opaqueFrame.OperandStack.PushInt(0)
	vmCtx.handleNativeCall("javax/microedition/lcdui/Image", "createRGBImage", "([III)Ljavax/microedition/lcdui/Image;", opaqueFrame, OpInvokestatic)

	opaqueImage, ok := opaqueFrame.OperandStack.PopRef().(*Object)
	if !ok || opaqueImage == nil {
		t.Fatal("createRGBImage did not return an image object")
	}
	opaqueRGBA, ok := opaqueImage.Extra.(*image.RGBA)
	if !ok {
		t.Fatal("createRGBImage did not return RGBA data")
	}
	_, _, _, alpha := opaqueRGBA.At(0, 0).RGBA()
	if alpha != 0xFFFF {
		t.Fatalf("processAlpha=false alpha = %#x, want opaque", alpha)
	}

	alphaFrame := &Frame{OperandStack: NewOperandStack(8)}
	alphaPixel := uint32(0x80112233)
	alphaFrame.OperandStack.PushRef([]int32{int32(alphaPixel)})
	alphaFrame.OperandStack.PushInt(1)
	alphaFrame.OperandStack.PushInt(1)
	alphaFrame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("javax/microedition/lcdui/Image", "createRGBImage", "([III)Ljavax/microedition/lcdui/Image;", alphaFrame, OpInvokestatic)

	alphaImage, ok := alphaFrame.OperandStack.PopRef().(*Object)
	if !ok || alphaImage == nil {
		t.Fatal("createRGBImage with alpha did not return an image object")
	}
	alphaRGBA, ok := alphaImage.Extra.(*image.RGBA)
	if !ok {
		t.Fatal("createRGBImage with alpha did not return RGBA data")
	}
	_, _, _, alpha = alphaRGBA.At(0, 0).RGBA()
	if alpha != 0x8080 {
		t.Fatalf("processAlpha=true alpha = %#x, want %#x", alpha, uint32(0x8080))
	}
}
