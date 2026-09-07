package vm

import (
	"encoding/binary"
	"os"
	"testing"

	"java-app/pkg/classfile"
	"java-app/pkg/jar"
	"java-app/pkg/midp"
)

func TestScanXORMethodsFindsAtLeastOne(t *testing.T) {
	b, err := os.ReadFile("../../mobile/assets/KnightOnline_402.jar")
	if err != nil {
		t.Skip(err)
	}
	found := ScanXORMethods(b)
	if len(found) == 0 {
		t.Fatal("expected at least one XOR stream-cipher method in KnightOnline_402.jar")
	}
}

func TestIdentityXORReturnsInputByte(t *testing.T) {
	if !IsIdentityXOR("synthetic/Xor", "a", "(B)B") && len(IdentityXORMethods) == 0 {
		// After scan, register. Direct unit: register then check.
	}
	RegisterIdentityXOR("synthetic/Xor.a:(B)B")
	if !IsIdentityXOR("synthetic/Xor", "a", "(B)B") {
		t.Fatal("expected registration")
	}

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	receiver := &Object{}
	frame.OperandStack.PushRef(receiver)
	frame.OperandStack.PushInt(0x5A)
	ctx.HandleNativeCall("synthetic/Xor", "a", "(B)B", frame, OpInvokevirtual)
	got := frame.OperandStack.PopInt()
	if got != 0x5A {
		t.Fatalf("identity XOR should return input byte unchanged, got %d", got)
	}
	if frame.OperandStack.size != 0 {
		t.Fatalf("expected empty stack after identity XOR, size=%d", frame.OperandStack.size)
	}
	if ctx.ActiveSocket != nil {
		t.Fatal("identity XOR must not touch SocketTap")
	}

	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushInt(0x7F)
	ctx.HandleNativeCall("synthetic/Xor", "a", "(B)B", frame, OpInvokestatic)
	got = frame.OperandStack.PopInt()
	if got != 0x7F {
		t.Fatalf("static identity XOR should return input byte unchanged, got %d", got)
	}

	// KnightOnline_402.jar read-side accessor is static l.a(Ll;B)B (field r).
	RegisterIdentityXOR("l.a:(Ll;B)B")
	if !IsIdentityXOR("l", "a", "(Ll;B)B") {
		t.Fatal("expected static accessor registration")
	}
	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(&Object{})
	frame.OperandStack.PushInt(0x3C)
	ctx.HandleNativeCall("l", "a", "(Ll;B)B", frame, OpInvokestatic)
	got = frame.OperandStack.PopInt()
	if got != 0x3C {
		t.Fatalf("static l.a(Ll;B)B should return input byte unchanged, got %d", got)
	}
	if frame.OperandStack.size != 0 {
		t.Fatalf("expected empty stack after static accessor XOR, size=%d", frame.OperandStack.size)
	}
}

// trapXorClassBytes is synthetic/Xor with a:(B)B whose body would return 0x11
// if the interpreter executed it instead of the identity stub.
func trapXorClassBytes() []byte {
	cp := [][]byte{
		append([]byte{classfile.TagUtf8, 0, byte(len("synthetic/Xor"))}, []byte("synthetic/Xor")...),
		{classfile.TagClass, 0, 1},
		append([]byte{classfile.TagUtf8, 0, byte(len("java/lang/Object"))}, []byte("java/lang/Object")...),
		{classfile.TagClass, 0, 3},
		append([]byte{classfile.TagUtf8, 0, byte(len("a"))}, []byte("a")...),
		append([]byte{classfile.TagUtf8, 0, byte(len("(B)B"))}, []byte("(B)B")...),
		append([]byte{classfile.TagUtf8, 0, byte(len("Code"))}, []byte("Code")...),
	}
	code := []byte{OpBipush, 0x11, OpIreturn}
	// Code attr: max_stack, max_locals, code_len, code, ex_len, sub_attr_count
	codeAttrInfo := []byte{0, 2, 0, 2, 0, 0, 0, byte(len(code))}
	codeAttrInfo = append(codeAttrInfo, code...)
	codeAttrInfo = append(codeAttrInfo, 0, 0, 0, 0)
	codeAttr := []byte{0, 7} // name index Code
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(codeAttrInfo)))
	codeAttr = append(codeAttr, lenBuf...)
	codeAttr = append(codeAttr, codeAttrInfo...)

	method := []byte{0, 1, 0, 5, 0, 6, 0, 1} // public, name a, desc (B)B, 1 attr
	method = append(method, codeAttr...)

	var out []byte
	out = append(out, 0xCA, 0xFE, 0xBA, 0xBE, 0, 0, 0, 49)
	out = append(out, 0, byte(len(cp)+1))
	for _, e := range cp {
		out = append(out, e...)
	}
	out = append(out, 0, 0x21) // ACC_PUBLIC|ACC_SUPER
	out = append(out, 0, 2)    // this = synthetic/Xor
	out = append(out, 0, 4)    // super = Object
	out = append(out, 0, 0)    // interfaces
	out = append(out, 0, 0)    // fields
	out = append(out, 0, 1)    // methods
	out = append(out, method...)
	out = append(out, 0, 0) // class attrs
	return out
}

func xorInvokeCallerCF() *classfile.ClassFile {
	return &classfile.ClassFile{
		ConstantPool: classfile.ConstantPool{
			nil,
			classfile.ConstantUtf8Info{Str: "synthetic/Xor"},
			classfile.ConstantClassInfo{NameIndex: 1},
			classfile.ConstantUtf8Info{Str: "a"},
			classfile.ConstantUtf8Info{Str: "(B)B"},
			classfile.ConstantNameAndTypeInfo{NameIndex: 3, DescriptorIndex: 4},
			classfile.ConstantMemberRefInfo{ClassIndex: 2, NameAndTypeIndex: 5},
		},
	}
}

func TestIdentityXORInvokeBytecodeDoesNotRunCallee(t *testing.T) {
	trap := trapXorClassBytes()
	cf, err := classfile.Parse(trap)
	if err != nil {
		t.Fatalf("trap class parse: %v", err)
	}
	m := cf.GetMethod("a", "(B)B")
	if m == nil || len(m.Code) == 0 {
		t.Fatal("trap class missing a:(B)B code")
	}

	RegisterIdentityXOR("synthetic/Xor.a:(B)B")
	loader := classfile.NewClassLoader(&jar.JarArchive{
		Classes: map[string][]byte{"synthetic/Xor": trap},
	})
	ctx := NewVMContext(loader, midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(&Object{})
	frame.OperandStack.PushInt(0x5A)
	code := []byte{OpInvokevirtual, 0x00, 0x06, OpIreturn}
	res, err := ctx.ExecuteBytecode(xorInvokeCallerCF(), codeWithFrame{code: code, frame: frame})
	if err != nil {
		t.Fatalf("ExecuteBytecode: %v", err)
	}
	if res.Num != 0x5A {
		t.Fatalf("invokevirtual identity XOR should return input 0x5A without running callee (0x11), got %d", res.Num)
	}
}
