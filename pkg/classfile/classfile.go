package classfile

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type FieldInfo struct {
	AccessFlags uint16
	NameIndex   uint16
	DescIndex   uint16
}

type ExceptionTableEntry struct {
	StartPC   uint16
	EndPC     uint16
	HandlerPC uint16
	CatchType uint16
}

type MethodInfo struct {
	AccessFlags    uint16
	NameIndex      uint16
	DescIndex      uint16
	Code           []byte
	MaxStack       uint16
	MaxLocals      uint16
	ExceptionTable []ExceptionTableEntry
}

type ClassFile struct {
	Magic        uint32
	MinorVersion uint16
	MajorVersion uint16
	ConstantPool ConstantPool
	AccessFlags  uint16
	ThisClass    uint16
	SuperClass   uint16
	Interfaces   []uint16
	Fields       []FieldInfo
	Methods      []MethodInfo
}

func (cf *ClassFile) GetClassName() string {
	return cf.ConstantPool.GetClassName(cf.ThisClass)
}

func (cf *ClassFile) GetSuperClassName() string {
	return cf.ConstantPool.GetClassName(cf.SuperClass)
}

func (cf *ClassFile) GetMethod(name, desc string) *MethodInfo {
	for i := range cf.Methods {
		mName := cf.ConstantPool.GetUtf8(cf.Methods[i].NameIndex)
		mDesc := cf.ConstantPool.GetUtf8(cf.Methods[i].DescIndex)
		if mName == name && (desc == "" || mDesc == desc) {
			return &cf.Methods[i]
		}
	}
	return nil
}

func Parse(data []byte) (*ClassFile, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("classfile data too short")
	}

	r := bytes.NewReader(data)

	cf := &ClassFile{}
	if err := binary.Read(r, binary.BigEndian, &cf.Magic); err != nil {
		return nil, err
	}
	if cf.Magic != 0xCAFEBABE {
		return nil, fmt.Errorf("invalid classfile magic: 0x%X", cf.Magic)
	}

	binary.Read(r, binary.BigEndian, &cf.MinorVersion)
	binary.Read(r, binary.BigEndian, &cf.MajorVersion)

	var cpCount uint16
	binary.Read(r, binary.BigEndian, &cpCount)

	cf.ConstantPool = make(ConstantPool, cpCount)
	for i := 1; i < int(cpCount); i++ {
		var tag uint8
		if err := binary.Read(r, binary.BigEndian, &tag); err != nil {
			return nil, err
		}
		switch tag {
		case TagUtf8:
			var length uint16
			binary.Read(r, binary.BigEndian, &length)
			strBytes := make([]byte, length)
			io.ReadFull(r, strBytes)
			cf.ConstantPool[i] = ConstantUtf8Info{Str: string(strBytes)}
		case TagInteger:
			var val int32
			binary.Read(r, binary.BigEndian, &val)
			cf.ConstantPool[i] = ConstantIntegerInfo{Value: val}
		case TagFloat:
			var val float32
			binary.Read(r, binary.BigEndian, &val)
			cf.ConstantPool[i] = ConstantFloatInfo{Value: val}
		case TagLong:
			var val int64
			binary.Read(r, binary.BigEndian, &val)
			cf.ConstantPool[i] = ConstantLongInfo{Value: val}
			i++ // Long takes two slots
		case TagDouble:
			var val float64
			binary.Read(r, binary.BigEndian, &val)
			cf.ConstantPool[i] = ConstantDoubleInfo{Value: val}
			i++ // Double takes two slots
		case TagClass:
			var nameIdx uint16
			binary.Read(r, binary.BigEndian, &nameIdx)
			cf.ConstantPool[i] = ConstantClassInfo{NameIndex: nameIdx}
		case TagString:
			var strIdx uint16
			binary.Read(r, binary.BigEndian, &strIdx)
			cf.ConstantPool[i] = ConstantStringInfo{StringIndex: strIdx}
		case TagFieldref, TagMethodref, TagInterfaceMethodref:
			var clsIdx, ntIdx uint16
			binary.Read(r, binary.BigEndian, &clsIdx)
			binary.Read(r, binary.BigEndian, &ntIdx)
			cf.ConstantPool[i] = ConstantMemberRefInfo{ClassIndex: clsIdx, NameAndTypeIndex: ntIdx}
		case TagNameAndType:
			var nameIdx, descIdx uint16
			binary.Read(r, binary.BigEndian, &nameIdx)
			binary.Read(r, binary.BigEndian, &descIdx)
			cf.ConstantPool[i] = ConstantNameAndTypeInfo{NameIndex: nameIdx, DescriptorIndex: descIdx}
		default:
			return nil, fmt.Errorf("unsupported constant pool tag %d at index %d", tag, i)
		}
	}

	binary.Read(r, binary.BigEndian, &cf.AccessFlags)
	binary.Read(r, binary.BigEndian, &cf.ThisClass)
	binary.Read(r, binary.BigEndian, &cf.SuperClass)

	var interfacesCount uint16
	binary.Read(r, binary.BigEndian, &interfacesCount)
	cf.Interfaces = make([]uint16, interfacesCount)
	for i := 0; i < int(interfacesCount); i++ {
		binary.Read(r, binary.BigEndian, &cf.Interfaces[i])
	}

	var fieldsCount uint16
	binary.Read(r, binary.BigEndian, &fieldsCount)
	cf.Fields = make([]FieldInfo, fieldsCount)
	for i := 0; i < int(fieldsCount); i++ {
		binary.Read(r, binary.BigEndian, &cf.Fields[i].AccessFlags)
		binary.Read(r, binary.BigEndian, &cf.Fields[i].NameIndex)
		binary.Read(r, binary.BigEndian, &cf.Fields[i].DescIndex)
		skipAttributes(r)
	}

	var methodsCount uint16
	binary.Read(r, binary.BigEndian, &methodsCount)
	cf.Methods = make([]MethodInfo, methodsCount)
	for i := 0; i < int(methodsCount); i++ {
		binary.Read(r, binary.BigEndian, &cf.Methods[i].AccessFlags)
		binary.Read(r, binary.BigEndian, &cf.Methods[i].NameIndex)
		binary.Read(r, binary.BigEndian, &cf.Methods[i].DescIndex)

		var attrCount uint16
		binary.Read(r, binary.BigEndian, &attrCount)
		for a := 0; a < int(attrCount); a++ {
			var attrNameIdx uint16
			var attrLen uint32
			binary.Read(r, binary.BigEndian, &attrNameIdx)
			binary.Read(r, binary.BigEndian, &attrLen)

			attrName := cf.ConstantPool.GetUtf8(attrNameIdx)
			if attrName == "Code" {
				binary.Read(r, binary.BigEndian, &cf.Methods[i].MaxStack)
				binary.Read(r, binary.BigEndian, &cf.Methods[i].MaxLocals)
				var codeLen uint32
				binary.Read(r, binary.BigEndian, &codeLen)
				cf.Methods[i].Code = make([]byte, codeLen)
				io.ReadFull(r, cf.Methods[i].Code)

				// Parse exception table
				var exCount uint16
				binary.Read(r, binary.BigEndian, &exCount)
				if exCount > 0 {
					cf.Methods[i].ExceptionTable = make([]ExceptionTableEntry, exCount)
					for e := 0; e < int(exCount); e++ {
						binary.Read(r, binary.BigEndian, &cf.Methods[i].ExceptionTable[e].StartPC)
						binary.Read(r, binary.BigEndian, &cf.Methods[i].ExceptionTable[e].EndPC)
						binary.Read(r, binary.BigEndian, &cf.Methods[i].ExceptionTable[e].HandlerPC)
						binary.Read(r, binary.BigEndian, &cf.Methods[i].ExceptionTable[e].CatchType)
					}
				}
				skipAttributes(r)
			} else {
				r.Seek(int64(attrLen), io.SeekCurrent)
			}
		}
	}

	return cf, nil
}

func skipAttributes(r *bytes.Reader) {
	var attrCount uint16
	if err := binary.Read(r, binary.BigEndian, &attrCount); err != nil {
		return
	}
	for a := 0; a < int(attrCount); a++ {
		var nameIdx uint16
		var length uint32
		binary.Read(r, binary.BigEndian, &nameIdx)
		binary.Read(r, binary.BigEndian, &length)
		r.Seek(int64(length), io.SeekCurrent)
	}
}
