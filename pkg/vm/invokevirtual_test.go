package vm

import (
	"encoding/binary"
	"testing"

	"java-app/pkg/classfile"
	"java-app/pkg/jar"
	"java-app/pkg/midp"
)

// simpleIntMethodClassBytes builds a public class with a single public int
// method whose body is `iconst_<ret>; ireturn`.
func simpleIntMethodClassBytes(thisName, superName, methodName string, ret byte) []byte {
	cp := [][]byte{
		append([]byte{classfile.TagUtf8, 0, byte(len(thisName))}, []byte(thisName)...),
		{classfile.TagClass, 0, 1},
		append([]byte{classfile.TagUtf8, 0, byte(len(superName))}, []byte(superName)...),
		{classfile.TagClass, 0, 3},
		append([]byte{classfile.TagUtf8, 0, byte(len(methodName))}, []byte(methodName)...),
		append([]byte{classfile.TagUtf8, 0, byte(len("()I"))}, []byte("()I")...),
		append([]byte{classfile.TagUtf8, 0, byte(len("Code"))}, []byte("Code")...),
	}
	code := []byte{ret, OpIreturn}
	codeAttrInfo := []byte{0, 2, 0, 2, 0, 0, 0, byte(len(code))}
	codeAttrInfo = append(codeAttrInfo, code...)
	codeAttrInfo = append(codeAttrInfo, 0, 0, 0, 0)
	codeAttr := []byte{0, 7}
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(codeAttrInfo)))
	codeAttr = append(codeAttr, lenBuf...)
	codeAttr = append(codeAttr, codeAttrInfo...)

	method := []byte{0, 1, 0, 5, 0, 6, 0, 1} // public, name, ()I, 1 attr
	method = append(method, codeAttr...)

	var out []byte
	out = append(out, 0xCA, 0xFE, 0xBA, 0xBE, 0, 0, 0, 49)
	out = append(out, 0, byte(len(cp)+1))
	for _, e := range cp {
		out = append(out, e...)
	}
	out = append(out, 0, 0x21) // ACC_PUBLIC|ACC_SUPER
	out = append(out, 0, 2)    // this
	out = append(out, 0, 4)    // super
	out = append(out, 0, 0)    // interfaces
	out = append(out, 0, 0)    // fields
	out = append(out, 0, 1)    // methods
	out = append(out, method...)
	out = append(out, 0, 0) // class attrs
	return out
}

func basePaintCallerCF() *classfile.ClassFile {
	return &classfile.ClassFile{
		ConstantPool: classfile.ConstantPool{
			nil,
			classfile.ConstantUtf8Info{Str: "Base"},
			classfile.ConstantClassInfo{NameIndex: 1},
			classfile.ConstantUtf8Info{Str: "paint"},
			classfile.ConstantUtf8Info{Str: "()I"},
			classfile.ConstantNameAndTypeInfo{NameIndex: 3, DescriptorIndex: 4},
			classfile.ConstantMemberRefInfo{ClassIndex: 2, NameAndTypeIndex: 5},
		},
	}
}

func TestInvokevirtualDispatchesToRuntimeOverride(t *testing.T) {
	baseBytes := simpleIntMethodClassBytes("Base", "java/lang/Object", "paint", OpIconst1)
	childBytes := simpleIntMethodClassBytes("Child", "Base", "paint", OpIconst2)

	baseCF, err := classfile.Parse(baseBytes)
	if err != nil {
		t.Fatalf("parse Base: %v", err)
	}
	childCF, err := classfile.Parse(childBytes)
	if err != nil {
		t.Fatalf("parse Child: %v", err)
	}
	if childCF.GetMethod("paint", "()I") == nil {
		t.Fatal("Child missing paint()I")
	}

	loader := classfile.NewClassLoader(&jar.JarArchive{
		Classes: map[string][]byte{
			"Base":  baseBytes,
			"Child": childBytes,
		},
	})
	ctx := NewVMContext(loader, midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(&Object{Class: childCF})
	code := []byte{OpInvokevirtual, 0x00, 0x06, OpIreturn}
	res, err := ctx.ExecuteBytecode(basePaintCallerCF(), codeWithFrame{code: code, frame: frame})
	if err != nil {
		t.Fatalf("ExecuteBytecode: %v", err)
	}
	if res.Num != 2 {
		t.Fatalf("invokevirtual Base.paint on Child must run Child.paint (2), got %d (Base.paint returns 1)", res.Num)
	}
	_ = baseCF
}

func TestInvokespecialUsesCompileTimeClass(t *testing.T) {
	baseBytes := simpleIntMethodClassBytes("Base", "java/lang/Object", "paint", OpIconst1)
	childBytes := simpleIntMethodClassBytes("Child", "Base", "paint", OpIconst2)

	childCF, err := classfile.Parse(childBytes)
	if err != nil {
		t.Fatalf("parse Child: %v", err)
	}

	loader := classfile.NewClassLoader(&jar.JarArchive{
		Classes: map[string][]byte{
			"Base":  baseBytes,
			"Child": childBytes,
		},
	})
	ctx := NewVMContext(loader, midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(&Object{Class: childCF})
	code := []byte{OpInvokespecial, 0x00, 0x06, OpIreturn}
	res, err := ctx.ExecuteBytecode(basePaintCallerCF(), codeWithFrame{code: code, frame: frame})
	if err != nil {
		t.Fatalf("ExecuteBytecode: %v", err)
	}
	if res.Num != 1 {
		t.Fatalf("invokespecial Base.paint on Child must run Base.paint (1), got %d", res.Num)
	}
}
