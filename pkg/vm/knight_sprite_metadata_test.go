package vm

import (
	"image/color"
	"os"
	"testing"

	"java-app/pkg/classfile"
	"java-app/pkg/jar"
	"java-app/pkg/midp"
)

func loadKnightSpriteVM(t *testing.T) (*VMContext, *classfile.ClassFile) {
	t.Helper()
	data, err := os.ReadFile("../../mobile/assets/KnightOnline_402.jar")
	if err != nil {
		t.Fatal(err)
	}
	archive, err := jar.ReadJar(data)
	if err != nil {
		t.Fatal(err)
	}
	loader := classfile.NewClassLoader(archive)
	ffClass, err := loader.LoadClass("ff")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewVMContext(loader, midp.NewGraphics(240, 320))
	ctx.SetStaticField("bx.b", Slot{Num: 1})
	ctx.SetStaticField("dx.j", Slot{Num: 1})
	return ctx, ffClass
}

func newKnightSprite(t *testing.T, ctx *VMContext, ffClass *classfile.ClassFile, layer, id int32) *Object {
	t.Helper()
	constructor := ffClass.GetMethod("<init>", "(BS)V")
	if constructor == nil {
		t.Fatal("Knight Online ff constructor missing")
	}
	sprite := NewObject(ffClass)
	if _, err := ctx.InvokeMethod(ffClass, constructor, []Slot{{Ref: sprite}, {Num: layer}, {Num: id}}); err != nil {
		t.Fatal(err)
	}
	return sprite
}

func TestKnightSpriteMetadataPreservesSignedOffsets(t *testing.T) {
	ctx, ffClass := loadKnightSpriteVM(t)
	sprite := newKnightSprite(t, ctx, ffClass, 1, 39)

	offsets, ok := sprite.GetField("j").Ref.([]Slot)
	if !ok || len(offsets) == 0 {
		t.Fatalf("body sprite offsets were not decoded: %#v", sprite.GetField("j").Ref)
	}
	firstDirection, ok := offsets[0].Ref.([]int32)
	if !ok || len(firstDirection) == 0 {
		t.Fatalf("body sprite first direction missing: %#v", offsets[0].Ref)
	}
	if got := firstDirection[0]; got != -9 {
		t.Fatalf("body b39 x offset = %d, want signed metadata value -9", got)
	}
}

func TestKnightBodySpriteDrawsFromBundledJar(t *testing.T) {
	ctx, ffClass := loadKnightSpriteVM(t)
	sprite := newKnightSprite(t, ctx, ffClass, 1, 39)
	drawMethod := ffClass.GetMethod("a", "(Lbx;IIII)V")
	if drawMethod == nil {
		t.Fatal("Knight Online ff draw method missing")
	}
	bxClass, err := ctx.ClassLoader.LoadClass("bx")
	if err != nil {
		t.Fatal(err)
	}
	graphicsWrapper := NewObject(bxClass)
	graphicsWrapper.SetField("a", Slot{Ref: ctx.Graphics})
	for part := int32(0); part < 5; part++ {
		if _, err := ctx.InvokeMethod(ffClass, drawMethod, []Slot{
			{Ref: sprite},
			{Ref: graphicsWrapper},
			{Num: 120},
			{Num: 160},
			{Num: 0},
			{Num: part},
		}); err != nil {
			t.Fatal(err)
		}
	}

	background := color.RGBA{R: 15, G: 25, B: 45, A: 255}
	changed := 0
	for y := 130; y < 190; y++ {
		for x := 90; x < 150; x++ {
			if ctx.Graphics.Buffer.RGBAAt(x, y) != background {
				changed++
			}
		}
	}
	if changed < 20 {
		t.Fatalf("bundled b39 body rendered only %d visible pixels", changed)
	}
}

func TestKnightPlayerLayerOrderInitializesCompletely(t *testing.T) {
	ctx, _ := loadKnightSpriteVM(t)
	faClass, err := ctx.ClassLoader.LoadClass("fa")
	if err != nil {
		t.Fatal(err)
	}
	ctx.EnsureClassInitialized(faClass)

	rows, ok := ctx.GetStaticField("fa.n").Ref.([]Slot)
	if !ok || len(rows) != 4 {
		t.Fatalf("fa.n layer rows = %#v, want four rows", ctx.GetStaticField("fa.n").Ref)
	}
	want := []byte{6, 0, 1, 2, 5, 4, 3, 6}
	for direction, rowSlot := range rows {
		row, ok := rowSlot.Ref.([]byte)
		if !ok {
			t.Fatalf("fa.n[%d] = %T, want []byte", direction, rowSlot.Ref)
		}
		if len(row) != len(want) {
			t.Fatalf("fa.n[%d] length = %d, want %d", direction, len(row), len(want))
		}
		for i := range want {
			if row[i] != want[i] {
				t.Fatalf("fa.n[%d][%d] = %d, want %d (row=%v)", direction, i, row[i], want[i], row)
			}
		}
	}
}

func TestKnightPlayerLayerSelectionUsesEquippedSprites(t *testing.T) {
	ctx, _ := loadKnightSpriteVM(t)
	faClass, err := ctx.ClassLoader.LoadClass("fa")
	if err != nil {
		t.Fatal(err)
	}
	ctx.EnsureClassInitialized(faClass)
	player := NewObject(faClass)
	player.SetField("cH", Slot{Num: 0})
	player.SetField("dG", Slot{Ref: []int16{-1, -1, -1, -1, -1, -1, -1}})
	player.SetField("bZ", Slot{Num: 39})
	player.SetField("ca", Slot{Num: 39})
	player.SetField("bX", Slot{Num: 39})
	player.SetField("bY", Slot{Num: 0})
	player.SetField("cb", Slot{Num: 11})
	player.SetField("cc", Slot{Num: 2})
	player.SetField("cd", Slot{Num: 21})

	selectLayer := faClass.GetMethod("p", "(I)I")
	if selectLayer == nil {
		t.Fatal("fa.p layer selector missing")
	}
	want := []int32{21, 39, 39, 0, 2, 11, 39, 21}
	for layerIndex, wantSprite := range want {
		got, err := ctx.InvokeMethod(faClass, selectLayer, []Slot{{Ref: player}, {Num: int32(layerIndex)}})
		if err != nil {
			t.Fatal(err)
		}
		if got.Num != wantSprite {
			t.Fatalf("fa.p(%d) = %d, want %d", layerIndex, got.Num, wantSprite)
		}
	}
}

func TestKnightMapPlayerDrawUsesEquippedBodyLayers(t *testing.T) {
	ctx, _ := loadKnightSpriteVM(t)
	faClass, err := ctx.ClassLoader.LoadClass("fa")
	if err != nil {
		t.Fatal(err)
	}
	ctx.EnsureClassInitialized(faClass)
	player := NewObject(faClass)
	constructor := faClass.GetMethod("<init>", "()V")
	if constructor == nil {
		t.Fatal("fa default constructor missing")
	}
	if _, err := ctx.InvokeMethod(faClass, constructor, []Slot{{Ref: player}}); err != nil {
		t.Fatal(err)
	}
	player.SetField("cH", Slot{Num: 0})
	player.SetField("cf", Slot{Num: 0})
	player.SetField("dG", Slot{Ref: []int16{-1, -1, -1, -1, -1, -1, -1}})
	player.SetField("bZ", Slot{Num: 39})
	player.SetField("ca", Slot{Num: 39})
	player.SetField("bX", Slot{Num: 39})
	player.SetField("bY", Slot{Num: 0})
	player.SetField("cb", Slot{Num: 11})
	player.SetField("cc", Slot{Num: 2})
	player.SetField("cd", Slot{Num: -1})
	player.SetField("dY", Slot{Num: -1})

	bxClass, err := ctx.ClassLoader.LoadClass("bx")
	if err != nil {
		t.Fatal(err)
	}
	graphicsWrapper := NewObject(bxClass)
	graphicsWrapper.SetField("a", Slot{Ref: ctx.Graphics})
	drawMethod := faClass.GetMethod("c", "(Lbx;III)V")
	if drawMethod == nil {
		t.Fatal("fa.c map draw method missing")
	}
	if _, err := ctx.InvokeMethod(faClass, drawMethod, []Slot{
		{Ref: player},
		{Ref: graphicsWrapper},
		{Num: 120},
		{Num: 170},
		{Num: 0},
	}); err != nil {
		t.Fatal(err)
	}

	redPixels := 0
	for y := 120; y < 200; y++ {
		for x := 80; x < 160; x++ {
			pixel := ctx.Graphics.Buffer.RGBAAt(x, y)
			if pixel.R > 100 && pixel.R > pixel.G*2 && pixel.R > pixel.B*2 {
				redPixels++
			}
		}
	}
	if redPixels < 10 {
		t.Fatalf("map player draw produced only %d red b39 pixels", redPixels)
	}
}
