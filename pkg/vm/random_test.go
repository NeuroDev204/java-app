package vm

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"java-app/pkg/classfile"
	"java-app/pkg/jar"
)

func seededRandom(t *testing.T, ctx *VMContext, seed int64) *Object {
	t.Helper()
	obj := NewObject(nil)
	frame := &Frame{OperandStack: NewOperandStack(4)}
	frame.OperandStack.PushRef(obj)
	frame.OperandStack.PushInt(int32(seed >> 32))
	frame.OperandStack.PushInt(int32(seed))
	ctx.handleNativeCall("java/util/Random", "<init>", "(J)V", frame, OpInvokespecial)
	return obj
}

func TestRandomNextIntJavaSequence(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	obj := seededRandom(t, ctx, 0)
	// Reference sequence from java.util.Random(0), including signed values.
	for _, want := range []int32{-1155484576, -723955400, 1033096058, -1690734402} {
		frame := &Frame{OperandStack: NewOperandStack(2)}
		frame.OperandStack.PushRef(obj)
		ctx.handleNativeCall("java/util/Random", "nextInt", "()I", frame, OpInvokevirtual)
		if got := frame.OperandStack.PopInt(); got != want {
			t.Fatalf("Random.nextInt() = %d, want %d", got, want)
		}
	}
}

func TestRandomBoundedJavaSequences(t *testing.T) {
	for _, test := range []struct {
		bound int32
		want  []int32
	}{
		{1, []int32{0, 0, 0, 0, 0, 0}},
		{16, []int32{5, 4, 15, 10, 10, 1}},
		{17, []int32{9, 15, 6, 11, 6, 2}},
		{1073741825, []int32{780201732, 624727082, 229364461, 400193852, 53587697, 791329214}},
	} {
		ctx := NewVMContext(nil, nil)
		obj := seededRandom(t, ctx, 0x123456789ABC)
		for _, want := range test.want {
			frame := &Frame{OperandStack: NewOperandStack(3)}
			frame.OperandStack.PushRef(obj)
			frame.OperandStack.PushInt(test.bound)
			ctx.handleNativeCall("java/util/Random", "nextInt", "(I)I", frame, OpInvokevirtual)
			if got := frame.OperandStack.PopInt(); got != want {
				t.Fatalf("nextInt(%d) = %d, want %d", test.bound, got, want)
			}
		}
	}
}

func TestRandomRejectsNonpositiveBound(t *testing.T) {
	for _, bound := range []int32{0, -1} {
		ctx := NewVMContext(nil, nil)
		obj := seededRandom(t, ctx, 0)
		frame := &Frame{OperandStack: NewOperandStack(3)}
		frame.OperandStack.PushRef(obj)
		frame.OperandStack.PushInt(bound)
		ctx.handleNativeCall("java/util/Random", "nextInt", "(I)I", frame, OpInvokevirtual)
		if frame.PendingException == nil || frame.PendingException.Class.GetClassName() != "java/lang/IllegalArgumentException" {
			t.Fatalf("nextInt(%d) did not throw IllegalArgumentException", bound)
		}
		if frame.OperandStack.size != 0 {
			t.Fatal("invalid bound left a value on the stack")
		}
	}
}

func TestRandomReseedAndNextLong(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	obj := seededRandom(t, ctx, 0)
	for i := 0; i < 2; i++ {
		frame := &Frame{OperandStack: NewOperandStack(4)}
		frame.OperandStack.PushRef(obj)
		ctx.handleNativeCall("java/util/Random", "nextLong", "()J", frame, OpInvokevirtual)
		low := uint32(frame.OperandStack.PopInt())
		high := frame.OperandStack.PopInt()
		if got := int64(high)<<32 | int64(low); got != -4962768465676381896 {
			t.Fatalf("nextLong() = %d", got)
		}
		frame.OperandStack.PushRef(obj)
		frame.OperandStack.PushInt(0)
		frame.OperandStack.PushInt(0)
		ctx.handleNativeCall("java/util/Random", "setSeed", "(J)V", frame, OpInvokevirtual)
	}
}

func TestKnightRandomHelperTerminates(t *testing.T) {
	// Run the real JAR helper in a subprocess: the broken native loops forever,
	// and must not leave a busy goroutine behind in the rest of the test suite.
	if os.Getenv("KNIGHT_RANDOM_HELPER_CHILD") != "1" {
		deadline, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(deadline, os.Args[0], "-test.run=^TestKnightRandomHelperTerminates$")
		cmd.Env = append(os.Environ(), "KNIGHT_RANDOM_HELPER_CHILD=1")
		output, err := cmd.CombinedOutput()
		if deadline.Err() != nil {
			t.Fatal("KnightOnline ak.g(3) never returned: game update cannot reach the next frame")
		}
		if err != nil {
			t.Fatalf("JAR helper: %v\n%s", err, output)
		}
		return
	}
	data, err := os.ReadFile("../../mobile/assets/KnightOnline_402.jar")
	if err != nil {
		t.Fatal(err)
	}
	archive, err := jar.ReadJar(data)
	if err != nil {
		t.Fatal(err)
	}
	loader := classfile.NewClassLoader(archive)
	cf, err := loader.LoadClass("ak")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewVMContext(loader, nil)
	// Only Random is needed by this helper; omit unrelated resource loading.
	ctx.InitializedClasses["ak"] = true
	ctx.SetStaticField("ak.g", Slot{Ref: seededRandom(t, ctx, 0)})
	method := cf.GetMethod("g", "(I)I")
	if method == nil {
		t.Fatal("JAR random helper missing")
	}
	result, err := ctx.InvokeMethod(cf, method, []Slot{{Num: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Num == 0 || result.Num <= -3 || result.Num >= 3 {
		t.Fatalf("unexpected nonzero random remainder: %d", result.Num)
	}
}
