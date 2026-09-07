package classfile


const (
	TagUtf8               = 1
	TagInteger            = 3
	TagFloat              = 4
	TagLong               = 5
	TagDouble             = 6
	TagClass              = 7
	TagString             = 8
	TagFieldref           = 9
	TagMethodref          = 10
	TagInterfaceMethodref = 11
	TagNameAndType        = 12
)

type ConstantInfo interface{}

type ConstantClassInfo struct {
	NameIndex uint16
}

type ConstantUtf8Info struct {
	Str string
}

type ConstantStringInfo struct {
	StringIndex uint16
}

type ConstantMemberRefInfo struct {
	ClassIndex       uint16
	NameAndTypeIndex uint16
}

type ConstantNameAndTypeInfo struct {
	NameIndex       uint16
	DescriptorIndex uint16
}

type ConstantIntegerInfo struct {
	Value int32
}

type ConstantFloatInfo struct {
	Value float32
}

type ConstantLongInfo struct {
	Value int64
}

type ConstantDoubleInfo struct {
	Value float64
}

type ConstantPool []ConstantInfo

func (cp ConstantPool) GetUtf8(index uint16) string {
	if int(index) < len(cp) {
		if utf8, ok := cp[index].(ConstantUtf8Info); ok {
			return utf8.Str
		}
	}
	return ""
}

func (cp ConstantPool) GetClassName(index uint16) string {
	if int(index) < len(cp) {
		if classInfo, ok := cp[index].(ConstantClassInfo); ok {
			return cp.GetUtf8(classInfo.NameIndex)
		}
	}
	return ""
}

func (cp ConstantPool) GetNameAndType(index uint16) (string, string) {
	if int(index) < len(cp) {
		if nt, ok := cp[index].(ConstantNameAndTypeInfo); ok {
			return cp.GetUtf8(nt.NameIndex), cp.GetUtf8(nt.DescriptorIndex)
		}
	}
	return "", ""
}

func (cp ConstantPool) GetMethodRef(index uint16) (string, string, string) {
	if int(index) < len(cp) {
		if ref, ok := cp[index].(ConstantMemberRefInfo); ok {
			className := cp.GetClassName(ref.ClassIndex)
			name, desc := cp.GetNameAndType(ref.NameAndTypeIndex)
			return className, name, desc
		}
	}
	return "", "", ""
}
