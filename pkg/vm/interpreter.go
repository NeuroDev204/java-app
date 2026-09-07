package vm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"java-app/pkg/classfile"
	"java-app/pkg/midp"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

type RecordStoreData struct {
	Records [][]byte
}

type VMContext struct {
	ClassLoader            *classfile.ClassLoader
	StaticFields           map[string]Slot
	Graphics               *midp.Graphics
	CurrentCanvas          *Object
	PendingTextBox         *Object
	InitializedClasses     map[string]bool
	RecordStores           map[string]*RecordStoreData
	ActiveSocket           *midp.SocketTap
	RepaintPending         bool
	debugAppearanceSynced  bool
	debugAppearanceWrites  int
	debugPlayerSpriteCalls int
	debugLayerSelections   int
	mu                     sync.RWMutex
	// execMu serializes top-level interpretation so Thread.start goroutines
	// cannot interleave with startApp/paint and leave game statics half-init.
	execMu sync.Mutex
}

// CallSerialized runs fn with exclusive VM execution. Nested InvokeMethod
// calls inside fn must not call CallSerialized again (non-reentrant).
func (ctx *VMContext) CallSerialized(fn func()) {
	if ctx == nil || fn == nil {
		return
	}
	ctx.execMu.Lock()
	defer ctx.execMu.Unlock()
	fn()
}

func (ctx *VMContext) ActiveTap() *midp.SocketTap {
	return ctx.ActiveSocket
}

func (ctx *VMContext) GetStaticField(key string) Slot {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	if val, ok := ctx.StaticFields[key]; ok {
		return val
	}
	dotKey := strings.ReplaceAll(key, "/", ".")
	if val, ok := ctx.StaticFields[dotKey]; ok {
		return val
	}
	slashKey := strings.ReplaceAll(key, ".", "/")
	if val, ok := ctx.StaticFields[slashKey]; ok {
		return val
	}
	if key == "java/lang/System.out" || dotKey == "java.lang.System.out" || key == "java/lang/System.err" || dotKey == "java.lang.System.err" {
		return Slot{Ref: &Object{Extra: "PrintStream"}}
	}
	return Slot{}
}

func (ctx *VMContext) SetStaticField(key string, val Slot) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.StaticFields == nil {
		ctx.StaticFields = make(map[string]Slot)
	}
	ctx.StaticFields[key] = val
	ctx.StaticFields[strings.ReplaceAll(key, "/", ".")] = val
	ctx.StaticFields[strings.ReplaceAll(key, ".", "/")] = val
}

// SyncPlayerAppearance ensures the entity rendered for the local player inherits
// the authentic equipment selected on the character screen.
func (ctx *VMContext) SyncPlayerAppearance() {
	cnGSlot := ctx.GetStaticField("cn.g")
	cnG, ok := cnGSlot.Ref.(*Object)
	if !ok || cnG == nil {
		return
	}

	charID := cnG.GetField("cu").Num
	charName := extractString(cnG.GetField("cC").Ref)
	activePlayer := cnG

	// Network parsing can materialize the local player twice. cn.j owns the
	// entity the map renderer traverses, so prefer its matching object and make
	// cn.g point to that same identity instead of dressing an invisible clone.
	cnJSlot := ctx.GetStaticField("cn.j")
	if cnJ, ok := cnJSlot.Ref.(*Object); ok && cnJ != nil {
		vSlot := cnJ.GetField("a")
		if vObj, ok := vSlot.Ref.(*Object); ok && vObj != nil {
			if vecData, ok := vObj.Extra.(*VectorData); ok && vecData != nil {
				foundExact := false
				var matchingRenderedPlayer *Object
				for i := 0; i < vecData.Size(); i++ {
					entity, isObject := vecData.Get(i).(*Object)
					if !isObject || entity == nil {
						continue
					}
					if entity == cnG {
						foundExact = true
					}
					entityID := entity.GetField("cu").Num
					entityName := extractString(entity.GetField("cC").Ref)
					if (charID != 0 && entityID == charID) || (charName != "" && strings.EqualFold(entityName, charName)) {
						matchingRenderedPlayer = entity
						if entity == cnG {
							break
						}
					}
				}
				if matchingRenderedPlayer != nil {
					activePlayer = matchingRenderedPlayer
					ctx.SetStaticField("cn.g", Slot{Ref: activePlayer})
				} else if !foundExact {
					vecData.Add(cnG)
				}
			}
		}
	}

	// Retrieve character selection list from x.a
	xASlot := ctx.GetStaticField("x.a")
	xA, ok := xASlot.Ref.(*Object)
	if !ok || xA == nil {
		return
	}

	vSlot := xA.GetField("a")
	vObj, ok := vSlot.Ref.(*Object)
	if !ok || vObj == nil {
		return
	}

	vecData, ok := vObj.Extra.(*VectorData)
	if !ok || vecData == nil || vecData.Size() == 0 {
		return
	}

	var targetBM *Object
	for i := 0; i < vecData.Size(); i++ {
		item := vecData.Get(i)
		if bmObj, ok := item.(*Object); ok && bmObj != nil {
			bmID := bmObj.GetField("cu").Num
			bmName := extractString(bmObj.GetField("cC").Ref)
			if (charID != 0 && bmID == charID) || (charName != "" && strings.EqualFold(bmName, charName)) {
				targetBM = bmObj
				break
			}
		}
	}
	if targetBM == nil {
		selectedIndex := int(ctx.GetStaticField("x.b").Num)
		if selectedIndex >= 0 && selectedIndex < vecData.Size() {
			if selectedCharacter, ok := vecData.Get(selectedIndex).(*Object); ok && selectedCharacter != nil {
				targetBM = selectedCharacter
			}
		}
	}

	if targetBM == nil {
		return
	}

	bZ := activePlayer.GetField("bZ").Num
	ca := activePlayer.GetField("ca").Num
	bX := activePlayer.GetField("bX").Num

	freshPlayerDefaults := bZ == 2 && ca == 2 && bX == 2 && activePlayer.GetField("cd").Num < 0 && activePlayer.GetField("ce").Num < 0
	if (bZ <= 0 && ca <= 0 && bX <= 0) || freshPlayerDefaults {
		bmBZ := targetBM.GetField("bZ")
		bmCA := targetBM.GetField("ca")
		bmBX := targetBM.GetField("bX")
		bmCD := targetBM.GetField("cd")
		bmCE := targetBM.GetField("ce")
		bmDY := targetBM.GetField("dY")

		if bmBZ.Num > 0 {
			activePlayer.SetField("bZ", bmBZ)
		}
		if bmCA.Num > 0 {
			activePlayer.SetField("ca", bmCA)
		}
		if bmBX.Num > 0 {
			activePlayer.SetField("bX", bmBX)
		}
		if bmCD.Num > 0 {
			activePlayer.SetField("cd", bmCD)
		}
		if bmCE.Num > 0 {
			activePlayer.SetField("ce", bmCE)
		}
		if bmDY.Num > 0 {
			activePlayer.SetField("dY", bmDY)
		}

		bmBY := targetBM.GetField("bY")
		bmCC := targetBM.GetField("cc")
		bmCB := targetBM.GetField("cb")
		bmClass := targetBM.GetField("by")
		bmLevel := targetBM.GetField("bz")
		bmCP := targetBM.GetField("cP")

		// fa.a(bx, int, int) renders these three layers directly. A fresh fa
		// instance already has non-zero defaults, so checking the map entity keeps
		// the wrong head/eye/hair instead of the selected character's layers.
		if targetBM.HasField("bY") {
			activePlayer.SetField("bY", bmBY)
		}
		if targetBM.HasField("cc") {
			activePlayer.SetField("cc", bmCC)
		}
		if targetBM.HasField("cb") {
			activePlayer.SetField("cb", bmCB)
		}
		if bmClass.Num > 0 && activePlayer.GetField("by").Num <= 0 {
			activePlayer.SetField("by", bmClass)
		}
		if bmLevel.Num > 0 && activePlayer.GetField("bz").Num <= 0 {
			activePlayer.SetField("bz", bmLevel)
		}
		if bmCP.Ref != nil && activePlayer.GetField("cP").Ref == nil {
			activePlayer.SetField("cP", bmCP)
		}
		log.Printf("[SYNC-PLAYER] Synchronized player '%s' (ID %d) appearance from character select: bZ=%d, ca=%d, bX=%d, cd=%d, dY=%d, cc=%d",
			charName, charID, activePlayer.GetField("bZ").Num, activePlayer.GetField("ca").Num, activePlayer.GetField("bX").Num, activePlayer.GetField("cd").Num, activePlayer.GetField("dY").Num, activePlayer.GetField("cc").Num)
		ctx.debugAppearanceSynced = true
		log.Printf("[DEBUG-APPEARANCE-SYNC] cn.g=%p active=%p name=%q id=%d", cnG, activePlayer, charName, charID)
	}
}

func (ctx *VMContext) EnsureClassInitialized(cf *classfile.ClassFile) {
	if cf == nil {
		return
	}
	className := cf.GetClassName()
	ctx.mu.Lock()
	if ctx.InitializedClasses == nil {
		ctx.InitializedClasses = make(map[string]bool)
	}
	if ctx.InitializedClasses[className] {
		ctx.mu.Unlock()
		return
	}
	ctx.InitializedClasses[className] = true
	ctx.mu.Unlock()

	clinit := cf.GetMethod("<clinit>", "()V")
	if clinit != nil {
		log.Printf("[CLINIT] Executing static initializer <clinit> for %s", className)
		ctx.InvokeMethod(cf, clinit, nil)
	}

	if className == "cf" {
		ctx.SetStaticField("cf.I", Slot{Num: 1})
		if mI := cf.GetMethod("i", "()V"); mI != nil {
			log.Printf("[TOUCH] Calling cf.i() to layout touch HUD")
			ctx.InvokeMethod(cf, mI, nil)
		}
	}
}

type DataStreamReader struct {
	Data []byte
	Pos  int
}

func (r *DataStreamReader) Available() int32 {
	rem := len(r.Data) - r.Pos
	if rem < 0 {
		return 0
	}
	return int32(rem)
}

func (r *DataStreamReader) ReadNextByte() byte {
	if r.Pos < len(r.Data) {
		b := r.Data[r.Pos]
		r.Pos++
		return b
	}
	return 0
}

func (r *DataStreamReader) ReadShort() int16 {
	if r.Pos+2 <= len(r.Data) {
		val := binary.BigEndian.Uint16(r.Data[r.Pos : r.Pos+2])
		r.Pos += 2
		return int16(val)
	}
	return 0
}

func (r *DataStreamReader) ReadInt() int32 {
	if r.Pos+4 <= len(r.Data) {
		val := binary.BigEndian.Uint32(r.Data[r.Pos : r.Pos+4])
		r.Pos += 4
		return int32(val)
	}
	return 0
}

func (r *DataStreamReader) ReadLong() int64 {
	if r.Pos+8 <= len(r.Data) {
		val := binary.BigEndian.Uint64(r.Data[r.Pos : r.Pos+8])
		r.Pos += 8
		return int64(val)
	}
	return 0
}

func (r *DataStreamReader) ReadBytes(dst []byte, off, length int) int32 {
	if r.Pos >= len(r.Data) || length <= 0 {
		return -1
	}
	n := length
	if r.Pos+n > len(r.Data) {
		n = len(r.Data) - r.Pos
	}
	if off < 0 || off >= len(dst) {
		return 0
	}
	toCopy := n
	if off+toCopy > len(dst) {
		toCopy = len(dst) - off
	}
	if toCopy > 0 {
		copy(dst[off:off+toCopy], r.Data[r.Pos:r.Pos+toCopy])
	}
	r.Pos += toCopy
	return int32(toCopy)
}

func (r *DataStreamReader) ReadUTF() string {
	if r.Pos+2 > len(r.Data) {
		return ""
	}
	utfLen := int(binary.BigEndian.Uint16(r.Data[r.Pos : r.Pos+2]))
	r.Pos += 2
	if r.Pos+utfLen > len(r.Data) {
		utfLen = len(r.Data) - r.Pos
	}
	if utfLen <= 0 {
		return ""
	}
	strVal := string(r.Data[r.Pos : r.Pos+utfLen])
	r.Pos += utfLen
	return strVal
}

func NewVMContext(cl *classfile.ClassLoader, g *midp.Graphics) *VMContext {
	ctx := &VMContext{
		ClassLoader:        cl,
		StaticFields:       make(map[string]Slot),
		Graphics:           g,
		InitializedClasses: make(map[string]bool),
		RecordStores:       make(map[string]*RecordStoreData),
	}
	printStreamObj := &Object{Extra: "PrintStream"}
	ctx.SetStaticField("java/lang/System.out", Slot{Ref: printStreamObj})
	ctx.SetStaticField("java/lang/System.err", Slot{Ref: printStreamObj})
	ctx.loadDefaultRMS()
	return ctx
}

func isGraphicsMethod(name string) bool {
	switch name {
	case "setColor", "fillRect", "drawRect", "drawLine", "drawString", "drawImage", "drawRegion", "drawRGB", "translate", "getTranslateX", "getTranslateY", "setClip", "clipRect", "getClipX", "getClipY", "getClipWidth", "getClipHeight":
		return true
	default:
		return false
	}
}

func countMethodParamSlots(desc string) int {
	if !strings.HasPrefix(desc, "(") {
		return 0
	}
	endIdx := strings.Index(desc, ")")
	if endIdx == -1 {
		return 0
	}
	paramsStr := desc[1:endIdx]
	count := 0
	i := 0
	for i < len(paramsStr) {
		c := paramsStr[i]
		if c == 'L' {
			count++
			semi := strings.IndexByte(paramsStr[i:], ';')
			if semi == -1 {
				break
			}
			i += semi + 1
		} else if c == '[' {
			i++
		} else {
			count++
			if c == 'J' || c == 'D' {
				count++
			}
			i++
		}
	}
	return count
}

func hasReturnValue(desc string) bool {
	endIdx := strings.Index(desc, ")")
	if endIdx != -1 && endIdx+1 < len(desc) {
		return desc[endIdx+1] != 'V'
	}
	return false
}

// resolveVirtualMethod walks the receiver's runtime class (then superclasses)
// for an override of mName+mDesc. invokevirtual / invokeinterface must use
// this instead of the compile-time class in the constant pool, otherwise
// Knight Online paint stays on p.a(Lbx;)V and never reaches u.a(Lbx;)V.
func (ctx *VMContext) resolveVirtualMethod(recv interface{}, mName, mDesc string) (*classfile.ClassFile, *classfile.MethodInfo) {
	obj, ok := recv.(*Object)
	if !ok || obj == nil || obj.Class == nil {
		return nil, nil
	}
	if m := obj.Class.GetMethod(mName, mDesc); m != nil && len(m.Code) > 0 {
		return obj.Class, m
	}
	if ctx.ClassLoader == nil {
		return nil, nil
	}
	name := obj.Class.GetSuperClassName()
	seen := map[string]bool{obj.Class.GetClassName(): true}
	for name != "" && name != "java/lang/Object" && !seen[name] {
		seen[name] = true
		cf, err := ctx.ClassLoader.LoadClass(name)
		if err != nil || cf == nil {
			break
		}
		if m := cf.GetMethod(mName, mDesc); m != nil && len(m.Code) > 0 {
			return cf, m
		}
		name = cf.GetSuperClassName()
	}
	return nil, nil
}

// ResolveVirtualMethod exposes virtual method lookup for bridge dispatch.
func (ctx *VMContext) ResolveVirtualMethod(recv interface{}, mName, mDesc string) (*classfile.ClassFile, *classfile.MethodInfo) {
	return ctx.resolveVirtualMethod(recv, mName, mDesc)
}

// adjustKnightOnlinePotionFallback keeps the JAR's local potion numbering and
// its image-record numbering aligned. The local path starts at potionIcon_01,
// while the bundled image records start at image4001; only bk.e -> bk.o is
// adjusted so item, NPC, monster, and other icon fallbacks remain unchanged.
func adjustKnightOnlinePotionFallback(
	callerClassName string,
	callerMethodName string,
	targetClassName string,
	targetMethodName string,
	targetDescriptor string,
	parameterSlots []Slot,
) {
	if callerClassName != "bk" || callerMethodName != "e" {
		return
	}
	if targetClassName != "bk" || targetMethodName != "o" || targetDescriptor != "(S)Laq;" {
		return
	}
	if len(parameterSlots) == 0 || parameterSlots[0].Num < 4000 {
		return
	}
	parameterSlots[0].Num++
}

func (ctx *VMContext) InvokeMethod(cf *classfile.ClassFile, method *classfile.MethodInfo, args []Slot) (res Slot, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("VM execution recover in method %s: %v", cf.ConstantPool.GetUtf8(method.NameIndex), r)
		}
	}()

	if method == nil || len(method.Code) == 0 {
		return Slot{}, nil
	}

	mName := cf.ConstantPool.GetUtf8(method.NameIndex)
	mDesc := cf.ConstantPool.GetUtf8(method.DescIndex)
	if mName == "startApp" {
		log.Printf("[METHOD-TRACE] Executing %s.%s%s", cf.GetClassName(), mName, mDesc)
	}

	maxLocals := uint(method.MaxLocals)
	if maxLocals < uint(len(args)) {
		maxLocals = uint(len(args))
	}
	maxStack := uint(method.MaxStack)
	if maxStack == 0 {
		maxStack = 16
	}

	frame := &Frame{
		LocalVars:    NewLocalVars(maxLocals),
		OperandStack: NewOperandStack(maxStack),
		PC:           0,
	}

	for i, arg := range args {
		frame.LocalVars.SetRef(uint(i), arg.Ref)
		if arg.Ref == nil && arg.Num != 0 {
			frame.LocalVars.SetInt(uint(i), arg.Num)
		}
	}

	isPaint := (mName == "paint") || strings.Contains(mDesc, "Graphics")
	res, err = ctx.ExecuteBytecode(cf, codeWithFrame{code: method.Code, frame: frame, isPaint: isPaint, method: method})
	return res, err
}

type JavaException struct {
	Object *Object
}

func (e *JavaException) Error() string {
	if e.Object != nil && e.Object.Class != nil {
		return fmt.Sprintf("java exception: %s", e.Object.Class.GetClassName())
	}
	return "java exception"
}

func (ctx *VMContext) NewException(className, msg string) *Object {
	var cf *classfile.ClassFile
	if ctx.ClassLoader != nil {
		cf, _ = ctx.ClassLoader.LoadClass(className)
	}
	if cf == nil {
		cp := make(classfile.ConstantPool, 3)
		cp[1] = classfile.ConstantClassInfo{NameIndex: 2}
		cp[2] = classfile.ConstantUtf8Info{Str: className}
		cf = &classfile.ClassFile{
			ConstantPool: cp,
			ThisClass:    1,
		}
	}
	ex := &Object{Class: cf, Fields: make(map[string]Slot)}
	if msg != "" {
		ex.Fields["detailMessage"] = Slot{Ref: msg}
	}
	return ex
}

func (ctx *VMContext) isSubclassOf(exCls, catchCls string) bool {
	if exCls == catchCls {
		return true
	}
	if catchCls == "java/lang/Object" || catchCls == "java/lang/Throwable" {
		return true
	}
	if catchCls == "java/lang/Exception" {
		if strings.HasSuffix(exCls, "Exception") || strings.HasSuffix(exCls, "Error") {
			return true
		}
	}
	if catchCls == "java/io/IOException" {
		if exCls == "java/io/EOFException" || exCls == "java/io/IOException" || exCls == "java/io/InterruptedIOException" {
			return true
		}
	}
	if ctx.ClassLoader != nil {
		cf, _ := ctx.ClassLoader.LoadClass(exCls)
		for cf != nil {
			super := cf.GetSuperClassName()
			if super == catchCls {
				return true
			}
			if super == "" || super == "java/lang/Object" {
				break
			}
			cf, _ = ctx.ClassLoader.LoadClass(super)
		}
	}
	return false
}

func (ctx *VMContext) checkException(cf *classfile.ClassFile, method *classfile.MethodInfo, frame *Frame, instrPC uint16) bool {
	if frame.PendingException == nil {
		return false
	}
	ex := frame.PendingException
	if method != nil && len(method.ExceptionTable) > 0 {
		for _, entry := range method.ExceptionTable {
			if instrPC >= entry.StartPC && instrPC < entry.EndPC {
				matches := false
				if entry.CatchType == 0 {
					matches = true
				} else if cf != nil {
					catchCls := cf.ConstantPool.GetClassName(entry.CatchType)
					exCls := ""
					if ex != nil && ex.Class != nil {
						exCls = ex.Class.GetClassName()
					}
					matches = ctx.isSubclassOf(exCls, catchCls)
				}
				if matches {
					frame.PendingException = nil
					frame.OperandStack.Clear()
					frame.OperandStack.PushRef(ex)
					frame.PC = int(entry.HandlerPC)
					return true
				}
			}
		}
	}
	return false
}

type codeWithFrame struct {
	code    []byte
	frame   *Frame
	isPaint bool
	method  *classfile.MethodInfo
}

func (ctx *VMContext) ExecuteBytecode(cf *classfile.ClassFile, cfFrame codeWithFrame) (Slot, error) {
	code := cfFrame.code
	frame := cfFrame.frame

	for frame.PC < len(code) {
		if frame.PendingException != nil {
			if ctx.checkException(cf, cfFrame.method, frame, uint16(frame.PC)) {
				continue
			}
			return Slot{}, &JavaException{Object: frame.PendingException}
		}

		instrPC := uint16(frame.PC)
		opcode := code[frame.PC]
		frame.PC++

		switch opcode {
		case OpNop:
		case OpAconstNull:
			frame.OperandStack.PushRef(nil)
		case OpIconstM1:
			frame.OperandStack.PushInt(-1)
		case OpIconst0:
			frame.OperandStack.PushInt(0)
		case OpIconst1:
			frame.OperandStack.PushInt(1)
		case OpIconst2:
			frame.OperandStack.PushInt(2)
		case OpIconst3:
			frame.OperandStack.PushInt(3)
		case OpIconst4:
			frame.OperandStack.PushInt(4)
		case OpIconst5:
			frame.OperandStack.PushInt(5)
		case OpBipush:
			val := int8(code[frame.PC])
			frame.PC++
			frame.OperandStack.PushInt(int32(val))
		case OpSipush:
			val := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			frame.PC += 2
			frame.OperandStack.PushInt(int32(val))
		case OpLdc:
			idx := code[frame.PC]
			frame.PC++
			ctx.pushConstant(cf, frame, uint16(idx))
		case OpLdcW, OpLdc2W:
			idx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			frame.PC += 2
			ctx.pushConstant(cf, frame, idx)
		case OpIload:
			idx := code[frame.PC]
			frame.PC++
			frame.OperandStack.PushInt(frame.LocalVars.GetInt(uint(idx)))
		case OpIload0:
			frame.OperandStack.PushInt(frame.LocalVars.GetInt(0))
		case OpIload1:
			frame.OperandStack.PushInt(frame.LocalVars.GetInt(1))
		case OpIload2:
			frame.OperandStack.PushInt(frame.LocalVars.GetInt(2))
		case OpIload3:
			frame.OperandStack.PushInt(frame.LocalVars.GetInt(3))
		case OpAload:
			idx := code[frame.PC]
			frame.PC++
			frame.OperandStack.PushRef(frame.LocalVars.GetRef(uint(idx)))
		case OpAload0:
			frame.OperandStack.PushRef(frame.LocalVars.GetRef(0))
		case OpAload1:
			frame.OperandStack.PushRef(frame.LocalVars.GetRef(1))
		case OpAload2:
			frame.OperandStack.PushRef(frame.LocalVars.GetRef(2))
		case OpAload3:
			frame.OperandStack.PushRef(frame.LocalVars.GetRef(3))
		case OpBaload:
			idx := frame.OperandStack.PopInt()
			arrRef := frame.OperandStack.PopRef()
			if b, ok := arrRef.([]byte); ok && idx >= 0 && int(idx) < len(b) {
				frame.OperandStack.PushInt(int32(int8(b[idx])))
			} else if obj, ok := arrRef.(*Object); ok && obj != nil {
				if b, ok := obj.Extra.([]byte); ok && idx >= 0 && int(idx) < len(b) {
					frame.OperandStack.PushInt(int32(int8(b[idx])))
				} else {
					frame.OperandStack.PushInt(0)
				}
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpCaload:
			idx := frame.OperandStack.PopInt()
			arrRef := frame.OperandStack.PopRef()
			if chars, ok := arrRef.([]uint16); ok && idx >= 0 && int(idx) < len(chars) {
				frame.OperandStack.PushInt(int32(chars[idx]))
			} else if b, ok := arrRef.([]byte); ok && idx >= 0 && int(idx) < len(b) {
				frame.OperandStack.PushInt(int32(b[idx]))
			} else if obj, ok := arrRef.(*Object); ok && obj != nil {
				if chars, ok := obj.Extra.([]uint16); ok && idx >= 0 && int(idx) < len(chars) {
					frame.OperandStack.PushInt(int32(chars[idx]))
				} else if b, ok := obj.Extra.([]byte); ok && idx >= 0 && int(idx) < len(b) {
					frame.OperandStack.PushInt(int32(b[idx]))
				} else {
					frame.OperandStack.PushInt(0)
				}
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpSaload:
			idx := frame.OperandStack.PopInt()
			arrRef := frame.OperandStack.PopRef()
			if shorts, ok := arrRef.([]int16); ok && idx >= 0 && int(idx) < len(shorts) {
				frame.OperandStack.PushInt(int32(shorts[idx]))
			} else if b, ok := arrRef.([]byte); ok && idx >= 0 && int(idx)*2+1 < len(b) {
				val := int16(binary.BigEndian.Uint16(b[idx*2:]))
				frame.OperandStack.PushInt(int32(val))
			} else if obj, ok := arrRef.(*Object); ok && obj != nil {
				if shorts, ok := obj.Extra.([]int16); ok && idx >= 0 && int(idx) < len(shorts) {
					frame.OperandStack.PushInt(int32(shorts[idx]))
				} else {
					frame.OperandStack.PushInt(0)
				}
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpIaload:
			idx := frame.OperandStack.PopInt()
			arrRef := frame.OperandStack.PopRef()
			if ints, ok := arrRef.([]int32); ok && idx >= 0 && int(idx) < len(ints) {
				frame.OperandStack.PushInt(ints[idx])
			} else if obj, ok := arrRef.(*Object); ok && obj != nil {
				if ints, ok := obj.Extra.([]int32); ok && idx >= 0 && int(idx) < len(ints) {
					frame.OperandStack.PushInt(ints[idx])
				} else {
					frame.OperandStack.PushInt(0)
				}
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpAaload:
			idx := frame.OperandStack.PopInt()
			arrRef := frame.OperandStack.PopRef()
			var slots []Slot
			var refs []interface{}
			if s, ok := arrRef.([]Slot); ok {
				slots = s
			} else if r, ok := arrRef.([]interface{}); ok {
				refs = r
			} else if obj, ok := arrRef.(*Object); ok && obj != nil {
				if s, ok := obj.Extra.([]Slot); ok {
					slots = s
				} else if r, ok := obj.Extra.([]interface{}); ok {
					refs = r
				}
			}

			if slots != nil && idx >= 0 && int(idx) < len(slots) {
				frame.OperandStack.PushInt(slots[idx].Num)
				if slots[idx].Ref != nil {
					frame.OperandStack.slots[frame.OperandStack.size-1].Ref = slots[idx].Ref
				}
			} else if refs != nil && idx >= 0 && int(idx) < len(refs) {
				frame.OperandStack.PushRef(refs[idx])
			} else {
				frame.OperandStack.PushRef(nil)
			}
		case OpIstore:
			idx := code[frame.PC]
			frame.PC++
			frame.LocalVars.SetInt(uint(idx), frame.OperandStack.PopInt())
		case OpIstore0:
			frame.LocalVars.SetInt(0, frame.OperandStack.PopInt())
		case OpIstore1:
			frame.LocalVars.SetInt(1, frame.OperandStack.PopInt())
		case OpIstore2:
			frame.LocalVars.SetInt(2, frame.OperandStack.PopInt())
		case OpIstore3:
			frame.LocalVars.SetInt(3, frame.OperandStack.PopInt())
		case OpAstore:
			idx := code[frame.PC]
			frame.PC++
			frame.LocalVars.SetRef(uint(idx), frame.OperandStack.PopRef())
		case OpAstore0:
			frame.LocalVars.SetRef(0, frame.OperandStack.PopRef())
		case OpAstore1:
			frame.LocalVars.SetRef(1, frame.OperandStack.PopRef())
		case OpAstore2:
			frame.LocalVars.SetRef(2, frame.OperandStack.PopRef())
		case OpAstore3:
			frame.LocalVars.SetRef(3, frame.OperandStack.PopRef())
		case OpIastore, OpBastore, OpCastore, OpSastore:
			val := frame.OperandStack.PopInt()
			idx := frame.OperandStack.PopInt()
			arrRef := frame.OperandStack.PopRef()
			if b, ok := arrRef.([]byte); ok && idx >= 0 && int(idx) < len(b) {
				b[idx] = byte(val)
			} else if ints, ok := arrRef.([]int32); ok && idx >= 0 && int(idx) < len(ints) {
				ints[idx] = val
			} else if shorts, ok := arrRef.([]int16); ok && idx >= 0 && int(idx) < len(shorts) {
				shorts[idx] = int16(val)
			} else if chars, ok := arrRef.([]uint16); ok && idx >= 0 && int(idx) < len(chars) {
				chars[idx] = uint16(val)
			} else if obj, ok := arrRef.(*Object); ok && obj != nil {
				if b, ok := obj.Extra.([]byte); ok && idx >= 0 && int(idx) < len(b) {
					b[idx] = byte(val)
				} else if ints, ok := obj.Extra.([]int32); ok && idx >= 0 && int(idx) < len(ints) {
					ints[idx] = val
				}
			}
		case OpAastore:
			if frame.OperandStack.size > 0 {
				valSlot := frame.OperandStack.slots[frame.OperandStack.size-1]
				frame.OperandStack.size--
				idx := frame.OperandStack.PopInt()
				arrRef := frame.OperandStack.PopRef()
				var slots []Slot
				var refs []interface{}
				if s, ok := arrRef.([]Slot); ok {
					slots = s
				} else if r, ok := arrRef.([]interface{}); ok {
					refs = r
				} else if obj, ok := arrRef.(*Object); ok && obj != nil {
					if s, ok := obj.Extra.([]Slot); ok {
						slots = s
					} else if r, ok := obj.Extra.([]interface{}); ok {
						refs = r
					}
				}

				if slots != nil && idx >= 0 && int(idx) < len(slots) {
					slots[idx] = valSlot
				} else if refs != nil && idx >= 0 && int(idx) < len(refs) {
					refs[idx] = valSlot.Ref
				}
			}
		case OpPop:
			frame.OperandStack.PopInt()
		case OpPop2:
			frame.OperandStack.PopInt()
			frame.OperandStack.PopInt()
		case OpDup:
			if frame.OperandStack.size > 0 {
				val := frame.OperandStack.slots[frame.OperandStack.size-1]
				frame.OperandStack.PushInt(val.Num)
				if val.Ref != nil {
					frame.OperandStack.slots[frame.OperandStack.size-1].Ref = val.Ref
				}
			}
		case OpLload:
			idx := uint(code[frame.PC])
			frame.PC++
			if int(idx)+1 < len(frame.LocalVars) {
				frame.OperandStack.PushSlot(frame.LocalVars[idx])
				frame.OperandStack.PushSlot(frame.LocalVars[idx+1])
			} else {
				frame.OperandStack.PushInt(0)
				frame.OperandStack.PushInt(0)
			}
		case 0x1E: // lload_0
			if 1 < len(frame.LocalVars) {
				frame.OperandStack.PushSlot(frame.LocalVars[0])
				frame.OperandStack.PushSlot(frame.LocalVars[1])
			} else {
				frame.OperandStack.PushInt(0)
				frame.OperandStack.PushInt(0)
			}
		case 0x1F: // lload_1
			if 2 < len(frame.LocalVars) {
				frame.OperandStack.PushSlot(frame.LocalVars[1])
				frame.OperandStack.PushSlot(frame.LocalVars[2])
			} else {
				frame.OperandStack.PushInt(0)
				frame.OperandStack.PushInt(0)
			}
		case 0x20: // lload_2
			if 3 < len(frame.LocalVars) {
				frame.OperandStack.PushSlot(frame.LocalVars[2])
				frame.OperandStack.PushSlot(frame.LocalVars[3])
			} else {
				frame.OperandStack.PushInt(0)
				frame.OperandStack.PushInt(0)
			}
		case 0x21: // lload_3
			if 4 < len(frame.LocalVars) {
				frame.OperandStack.PushSlot(frame.LocalVars[3])
				frame.OperandStack.PushSlot(frame.LocalVars[4])
			} else {
				frame.OperandStack.PushInt(0)
				frame.OperandStack.PushInt(0)
			}
		case OpLstore:
			idx := uint(code[frame.PC])
			frame.PC++
			if int(idx)+1 < len(frame.LocalVars) {
				frame.LocalVars[idx+1] = frame.OperandStack.PopSlot()
				frame.LocalVars[idx] = frame.OperandStack.PopSlot()
			} else {
				frame.OperandStack.PopSlot()
				frame.OperandStack.PopSlot()
			}
		case 0x3F: // lstore_0
			if 1 < len(frame.LocalVars) {
				frame.LocalVars[1] = frame.OperandStack.PopSlot()
				frame.LocalVars[0] = frame.OperandStack.PopSlot()
			} else {
				frame.OperandStack.PopSlot()
				frame.OperandStack.PopSlot()
			}
		case 0x40: // lstore_1
			if 2 < len(frame.LocalVars) {
				frame.LocalVars[2] = frame.OperandStack.PopSlot()
				frame.LocalVars[1] = frame.OperandStack.PopSlot()
			} else {
				frame.OperandStack.PopSlot()
				frame.OperandStack.PopSlot()
			}
		case 0x41: // lstore_2
			if 3 < len(frame.LocalVars) {
				frame.LocalVars[3] = frame.OperandStack.PopSlot()
				frame.LocalVars[2] = frame.OperandStack.PopSlot()
			} else {
				frame.OperandStack.PopSlot()
				frame.OperandStack.PopSlot()
			}
		case 0x42: // lstore_3
			if 4 < len(frame.LocalVars) {
				frame.LocalVars[4] = frame.OperandStack.PopSlot()
				frame.LocalVars[3] = frame.OperandStack.PopSlot()
			} else {
				frame.OperandStack.PopSlot()
				frame.OperandStack.PopSlot()
			}
		case OpDup2:
			if frame.OperandStack.size >= 2 {
				s2 := frame.OperandStack.slots[frame.OperandStack.size-1]
				s1 := frame.OperandStack.slots[frame.OperandStack.size-2]
				frame.OperandStack.PushSlot(s1)
				frame.OperandStack.PushSlot(s2)
			}
		case 0x61: // ladd
			s2_low := uint32(frame.OperandStack.PopInt())
			s2_high := uint32(frame.OperandStack.PopInt())
			s1_low := uint32(frame.OperandStack.PopInt())
			s1_high := uint32(frame.OperandStack.PopInt())
			v2 := (int64(s2_high) << 32) | int64(s2_low)
			v1 := (int64(s1_high) << 32) | int64(s1_low)
			res := v1 + v2
			frame.OperandStack.PushInt(int32(res >> 32))
			frame.OperandStack.PushInt(int32(res))
		case 0x65: // lsub
			s2_low := uint32(frame.OperandStack.PopInt())
			s2_high := uint32(frame.OperandStack.PopInt())
			s1_low := uint32(frame.OperandStack.PopInt())
			s1_high := uint32(frame.OperandStack.PopInt())
			v2 := (int64(s2_high) << 32) | int64(s2_low)
			v1 := (int64(s1_high) << 32) | int64(s1_low)
			res := v1 - v2
			frame.OperandStack.PushInt(int32(res >> 32))
			frame.OperandStack.PushInt(int32(res))
		case 0x94: // lcmp
			s2_low := uint32(frame.OperandStack.PopInt())
			s2_high := uint32(frame.OperandStack.PopInt())
			s1_low := uint32(frame.OperandStack.PopInt())
			s1_high := uint32(frame.OperandStack.PopInt())
			v2 := (int64(s2_high) << 32) | int64(s2_low)
			v1 := (int64(s1_high) << 32) | int64(s1_low)
			if v1 > v2 {
				frame.OperandStack.PushInt(1)
			} else if v1 < v2 {
				frame.OperandStack.PushInt(-1)
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpSwap:
			v1 := frame.OperandStack.PopInt()
			v2 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1)
			frame.OperandStack.PushInt(v2)
		case OpIadd:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1 + v2)
		case OpIsub:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1 - v2)
		case OpImul:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1 * v2)
		case OpIdiv:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			if v2 != 0 {
				frame.OperandStack.PushInt(v1 / v2)
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpIrem:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			if v2 != 0 {
				frame.OperandStack.PushInt(v1 % v2)
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpIneg:
			v := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(-v)
		case OpIshl:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			s := uint32(v2) & 0x1f
			frame.OperandStack.PushInt(v1 << s)
		case OpIshr:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			s := uint32(v2) & 0x1f
			frame.OperandStack.PushInt(v1 >> s)
		case OpIushr:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			s := uint32(v2) & 0x1f
			frame.OperandStack.PushInt(int32(uint32(v1) >> s))
		case OpIand:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1 & v2)
		case OpIor:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1 | v2)
		case OpIxor:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1 ^ v2)
		case OpIinc:
			idx := uint(code[frame.PC])
			constVal := int32(int8(code[frame.PC+1]))
			frame.PC += 2
			frame.LocalVars.SetInt(idx, frame.LocalVars.GetInt(idx)+constVal)
		case OpGoto:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			frame.PC += int(offset) - 1
		case OpIeq:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			slot := frame.OperandStack.PopSlot()
			if slot.Num == 0 && slot.Ref == nil {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIne:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			slot := frame.OperandStack.PopSlot()
			if slot.Num != 0 || slot.Ref != nil {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIlt:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v := frame.OperandStack.PopInt()
			if v < 0 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIge:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v := frame.OperandStack.PopInt()
			if v >= 0 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIgt:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v := frame.OperandStack.PopInt()
			if v > 0 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIle:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v := frame.OperandStack.PopInt()
			if v <= 0 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfIcmpeq:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			s2 := frame.OperandStack.PopSlot()
			s1 := frame.OperandStack.PopSlot()
			isEqual := false
			if s1.Ref != nil || s2.Ref != nil {
				isEqual = (s1.Ref == s2.Ref)
			} else {
				isEqual = (s1.Num == s2.Num)
			}
			if isEqual {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfIcmpne:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			s2 := frame.OperandStack.PopSlot()
			s1 := frame.OperandStack.PopSlot()
			isEqual := false
			if s1.Ref != nil || s2.Ref != nil {
				isEqual = (s1.Ref == s2.Ref)
			} else {
				isEqual = (s1.Num == s2.Num)
			}
			if !isEqual {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfIcmplt:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			if v1 < v2 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfIcmpge:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			if v1 >= v2 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfIcmpgt:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			if v1 > v2 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfIcmple:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			if v1 <= v2 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfAcmpeq:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			r2 := frame.OperandStack.PopRef()
			r1 := frame.OperandStack.PopRef()
			if r1 == r2 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfAcmpne:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			r2 := frame.OperandStack.PopRef()
			r1 := frame.OperandStack.PopRef()
			if r1 != r2 {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpTableswitch:
			startPC := frame.PC - 1
			pad := (4 - ((startPC + 1) % 4)) % 4
			frame.PC += pad
			defaultOffset := int32(binary.BigEndian.Uint32(code[frame.PC : frame.PC+4]))
			frame.PC += 4
			low := int32(binary.BigEndian.Uint32(code[frame.PC : frame.PC+4]))
			frame.PC += 4
			high := int32(binary.BigEndian.Uint32(code[frame.PC : frame.PC+4]))
			frame.PC += 4
			count := high - low + 1
			offsets := make([]int32, count)
			for i := int32(0); i < count; i++ {
				offsets[i] = int32(binary.BigEndian.Uint32(code[frame.PC : frame.PC+4]))
				frame.PC += 4
			}
			index := frame.OperandStack.PopInt()
			if index < low || index > high {
				frame.PC = startPC + int(defaultOffset)
			} else {
				frame.PC = startPC + int(offsets[index-low])
			}
		case OpI2b:
			val := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(int32(int8(val)))
		case OpI2c:
			val := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(int32(uint16(val)))
		case OpI2s:
			val := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(int32(int16(val)))
		case OpI2l, OpL2i, OpI2f, OpI2d, OpF2i, OpD2i:
			// Type conversion stubs
		case OpLookupswitch:
			startPC := frame.PC - 1
			pad := (4 - ((startPC + 1) % 4)) % 4
			frame.PC += pad
			defaultOffset := int32(binary.BigEndian.Uint32(code[frame.PC : frame.PC+4]))
			frame.PC += 4
			npairs := int32(binary.BigEndian.Uint32(code[frame.PC : frame.PC+4]))
			frame.PC += 4
			key := frame.OperandStack.PopInt()
			jumpOffset := defaultOffset
			found := false
			for i := int32(0); i < npairs; i++ {
				matchVal := int32(binary.BigEndian.Uint32(code[frame.PC : frame.PC+4]))
				offsetVal := int32(binary.BigEndian.Uint32(code[frame.PC+4 : frame.PC+8]))
				frame.PC += 8
				if !found && key == matchVal {
					jumpOffset = offsetVal
					found = true
				}
			}
			frame.PC = startPC + int(jumpOffset)
		case OpInstanceof:
			frame.PC += 2
			objRef := frame.OperandStack.PopRef()
			if objRef != nil {
				frame.OperandStack.PushInt(1)
			} else {
				frame.OperandStack.PushInt(0)
			}
		case OpCheckcast:
			frame.PC += 2
		case OpMonitorenter, OpMonitorexit:
			_ = frame.OperandStack.PopRef()
		case OpIfnull:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			ref := frame.OperandStack.PopRef()
			if ref == nil {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpIfnonnull:
			offset := int16(binary.BigEndian.Uint16(code[frame.PC : frame.PC+2]))
			ref := frame.OperandStack.PopRef()
			if ref != nil {
				frame.PC += int(offset) - 1
			} else {
				frame.PC += 2
			}
		case OpReturn:
			if cf != nil && cf.GetClassName() == "er" && cfFrame.method != nil && cf.ConstantPool.GetUtf8(cfFrame.method.NameIndex) == "b" {
				ctx.SyncPlayerAppearance()
			}
			return Slot{}, nil
		case OpIreturn:
			if ctx.debugAppearanceSynced && ctx.debugLayerSelections < 80 && cf != nil && cf.GetClassName() == "fa" && cfFrame.method != nil && cf.ConstantPool.GetUtf8(cfFrame.method.NameIndex) == "p" {
				cnG, _ := ctx.GetStaticField("cn.g").Ref.(*Object)
				if frame.LocalVars.GetRef(0) == cnG && frame.OperandStack.size > 0 {
					ctx.debugLayerSelections++
					log.Printf("[DEBUG-LAYER-SELECT] index=%d result=%d dG=%v", frame.LocalVars.GetInt(1), frame.OperandStack.slots[frame.OperandStack.size-1].Num, cnG.GetField("dG").Ref)
				}
			}
			return Slot{Num: frame.OperandStack.PopInt()}, nil
		case OpAreturn:
			return Slot{Ref: frame.OperandStack.PopRef()}, nil
		case OpNew:
			clsIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			frame.PC += 2
			clsName := cf.ConstantPool.GetClassName(clsIdx)
			targetCf, _ := ctx.ClassLoader.LoadClass(clsName)
			ctx.EnsureClassInitialized(targetCf)
			obj := NewObject(targetCf)
			frame.OperandStack.PushRef(obj)
		case OpNewarray:
			atype := code[frame.PC]
			frame.PC++
			size := frame.OperandStack.PopInt()
			if size < 0 {
				size = 0
			}
			n := int(size)
			// JVM atypes: 4 boolean, 5 char, 6 float, 7 double, 8 byte, 9 short, 10 int, 11 long.
			// Font metrics in Knight Online are int[4] per glyph; creating []byte made iaload return 0
			// and drawRegion text with src(0,0,0,0).
			switch atype {
			case 5:
				frame.OperandStack.PushRef(make([]uint16, n))
			case 6:
				frame.OperandStack.PushRef(make([]float32, n))
			case 7:
				frame.OperandStack.PushRef(make([]float64, n))
			case 9:
				frame.OperandStack.PushRef(make([]int16, n))
			case 10:
				frame.OperandStack.PushRef(make([]int32, n))
			case 11:
				frame.OperandStack.PushRef(make([]int64, n))
			default:
				frame.OperandStack.PushRef(make([]byte, n))
			}
		case OpAnewarray:
			clsIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			frame.PC += 2
			_ = clsIdx
			size := frame.OperandStack.PopInt()
			if size < 0 {
				size = 0
			}
			frame.OperandStack.PushRef(make([]Slot, size))
		case OpMultianewarray:
			clsIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			dims := code[frame.PC+2]
			frame.PC += 3
			desc := cf.ConstantPool.GetClassName(clsIdx)
			counts := make([]int, dims)
			for i := int(dims) - 1; i >= 0; i-- {
				counts[i] = int(frame.OperandStack.PopInt())
				if counts[i] < 0 {
					counts[i] = 0
				}
			}
			frame.OperandStack.PushRef(createMultiArray(desc, counts))
		case OpArraylength:
			arrRef := frame.OperandStack.PopRef()
			switch a := arrRef.(type) {
			case []byte:
				frame.OperandStack.PushInt(int32(len(a)))
			case []Slot:
				frame.OperandStack.PushInt(int32(len(a)))
			case []int32:
				frame.OperandStack.PushInt(int32(len(a)))
			case []int16:
				frame.OperandStack.PushInt(int32(len(a)))
			case []uint16:
				frame.OperandStack.PushInt(int32(len(a)))
			case []int64:
				frame.OperandStack.PushInt(int32(len(a)))
			case []interface{}:
				frame.OperandStack.PushInt(int32(len(a)))
			case *Object:
				if a != nil {
					switch extra := a.Extra.(type) {
					case []byte:
						frame.OperandStack.PushInt(int32(len(extra)))
					case []Slot:
						frame.OperandStack.PushInt(int32(len(extra)))
					case []int32:
						frame.OperandStack.PushInt(int32(len(extra)))
					case []int16:
						frame.OperandStack.PushInt(int32(len(extra)))
					case []uint16:
						frame.OperandStack.PushInt(int32(len(extra)))
					case []int64:
						frame.OperandStack.PushInt(int32(len(extra)))
					case []interface{}:
						frame.OperandStack.PushInt(int32(len(extra)))
					default:
						frame.OperandStack.PushInt(0)
					}
				} else {
					frame.OperandStack.PushInt(0)
				}
			default:
				frame.OperandStack.PushInt(0)
			}
		case OpGetstatic:
			refIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			frame.PC += 2
			if member, ok := cf.ConstantPool[refIdx].(classfile.ConstantMemberRefInfo); ok {
				clsName := cf.ConstantPool.GetClassName(member.ClassIndex)
				targetCf, _ := ctx.ClassLoader.LoadClass(clsName)
				ctx.EnsureClassInitialized(targetCf)
				fName, fDesc := cf.ConstantPool.GetNameAndType(member.NameAndTypeIndex)
				val := ctx.GetStaticField(clsName + "." + fName)
				if clsName == "cf" && fName == "I" {
					val.Num = 1
				} else if clsName == "fu" && fName == "D" {
					val.Num = 1
				} else if clsName == "cg" && fName == "s" {
					val.Ref = nil
					val.Num = 0
				} else if val.Num <= 0 {
					if (fName == "X" && clsName == "fu") || (fName == "a" && clsName == "com/silverknight/a") {
						val.Num = 240
					} else if (fName == "Y" && clsName == "fu") || (fName == "b" && clsName == "com/silverknight/a") {
						val.Num = 320
					}
				}
				if fDesc == "J" || fDesc == "D" {
					frame.OperandStack.PushInt(int32(int64(val.Num) >> 32))
					frame.OperandStack.PushInt(int32(val.Num))
				} else {
					frame.OperandStack.PushInt(val.Num)
					if val.Ref != nil {
						frame.OperandStack.slots[frame.OperandStack.size-1].Ref = val.Ref
					}
				}
			}
		case OpPutstatic:
			refIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			frame.PC += 2
			if member, ok := cf.ConstantPool[refIdx].(classfile.ConstantMemberRefInfo); ok {
				clsName := cf.ConstantPool.GetClassName(member.ClassIndex)
				targetCf, _ := ctx.ClassLoader.LoadClass(clsName)
				ctx.EnsureClassInitialized(targetCf)
				fName, fDesc := cf.ConstantPool.GetNameAndType(member.NameAndTypeIndex)
				if fDesc == "J" || fDesc == "D" {
					lowSlot := frame.OperandStack.PopSlot()
					highSlot := frame.OperandStack.PopSlot()
					combined := (int64(highSlot.Num) << 32) | (int64(lowSlot.Num) & 0xFFFFFFFF)
					ctx.SetStaticField(clsName+"."+fName, Slot{Num: int32(combined)})
				} else {
					if frame.OperandStack.size > 0 {
						topSlot := frame.OperandStack.PopSlot()
						if clsName == "cf" && fName == "I" {
							topSlot.Num = 1
						}
						ctx.SetStaticField(clsName+"."+fName, topSlot)
					}
				}
			}
		case OpGetfield:
			refIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			frame.PC += 2
			if member, ok := cf.ConstantPool[refIdx].(classfile.ConstantMemberRefInfo); ok {
				fName, fDesc := cf.ConstantPool.GetNameAndType(member.NameAndTypeIndex)
				objRef := frame.OperandStack.PopRef()
				if obj, ok := objRef.(*Object); ok && obj != nil {
					val := obj.GetField(fName)
					if fDesc == "J" || fDesc == "D" {
						frame.OperandStack.PushInt(int32(int64(val.Num) >> 32))
						frame.OperandStack.PushInt(int32(val.Num))
					} else {
						frame.OperandStack.PushInt(val.Num)
						if val.Ref != nil {
							frame.OperandStack.slots[frame.OperandStack.size-1].Ref = val.Ref
						}
					}
				} else {
					if fDesc == "J" || fDesc == "D" {
						frame.OperandStack.PushInt(0)
						frame.OperandStack.PushInt(0)
					} else {
						frame.OperandStack.PushRef(nil)
					}
				}
			}
		case OpPutfield:
			refIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			frame.PC += 2
			if member, ok := cf.ConstantPool[refIdx].(classfile.ConstantMemberRefInfo); ok {
				fName, fDesc := cf.ConstantPool.GetNameAndType(member.NameAndTypeIndex)
				if fDesc == "J" || fDesc == "D" {
					lowSlot := frame.OperandStack.PopSlot()
					highSlot := frame.OperandStack.PopSlot()
					objRef := frame.OperandStack.PopRef()
					if obj, ok := objRef.(*Object); ok && obj != nil {
						combined := (int64(highSlot.Num) << 32) | (int64(lowSlot.Num) & 0xFFFFFFFF)
						obj.SetField(fName, Slot{Num: int32(combined)})
					}
				} else {
					if frame.OperandStack.size > 0 {
						valSlot := frame.OperandStack.PopSlot()
						objRef := frame.OperandStack.PopRef()
						if obj, ok := objRef.(*Object); ok && obj != nil {
							if ctx.debugAppearanceSynced && ctx.debugAppearanceWrites < 100 &&
								(fName == "bZ" || fName == "ca" || fName == "bX" || fName == "cd" || fName == "ce" || fName == "dY" || fName == "bY" || fName == "cb" || fName == "cc") {
								name := extractString(obj.GetField("cC").Ref)
								cnG, _ := ctx.GetStaticField("cn.g").Ref.(*Object)
								if obj == cnG || strings.EqualFold(name, "vjncible") {
									callerMethod := ""
									if cfFrame.method != nil {
										callerMethod = cf.ConstantPool.GetUtf8(cfFrame.method.NameIndex)
									}
									ctx.debugAppearanceWrites++
									log.Printf("[DEBUG-APPEARANCE-WRITE] caller=%s.%s object=%p is-cn.g=%t name=%q field=%s old=%d new=%d",
										cf.GetClassName(), callerMethod, obj, obj == cnG, name, fName, obj.GetField(fName).Num, valSlot.Num)
								}
							}
							obj.SetField(fName, valSlot)
						}
					}
				}
			}
		case OpInvokestatic, OpInvokespecial, OpInvokevirtual, OpInvokeinterface:
			refIdx := binary.BigEndian.Uint16(code[frame.PC : frame.PC+2])
			if opcode == OpInvokeinterface {
				frame.PC += 4
			} else {
				frame.PC += 2
			}
			if member, ok := cf.ConstantPool[refIdx].(classfile.ConstantMemberRefInfo); ok {
				clsName := cf.ConstantPool.GetClassName(member.ClassIndex)
				mName, mDesc := cf.ConstantPool.GetNameAndType(member.NameAndTypeIndex)

				if clsName == "javax/microedition/lcdui/Graphics" || isGraphicsMethodName(mName) {
					ctx.handleNativeCall("javax/microedition/lcdui/Graphics", mName, mDesc, frame, opcode)
				} else if strings.HasPrefix(clsName, "javax/microedition/") || strings.HasPrefix(clsName, "java/") {
					ctx.handleNativeCall(clsName, mName, mDesc, frame, opcode)
				} else if IsIdentityXOR(clsName, mName, mDesc) {
					ctx.handleNativeCall(clsName, mName, mDesc, frame, opcode)
				} else {
					targetCf, _ := ctx.ClassLoader.LoadClass(clsName)
					if targetCf != nil {
						method := targetCf.GetMethod(mName, mDesc)
						if opcode == OpInvokevirtual || opcode == OpInvokeinterface {
							paramCount := countMethodParamSlots(mDesc)
							if frame.OperandStack.size > uint(paramCount) {
								recv := frame.OperandStack.slots[frame.OperandStack.size-1-uint(paramCount)].Ref
								if cf, m := ctx.resolveVirtualMethod(recv, mName, mDesc); m != nil {
									targetCf, method = cf, m
								}
							}
						}
						if method != nil {
							paramCount := countMethodParamSlots(mDesc)
							paramSlots := make([]Slot, paramCount)
							for i := paramCount - 1; i >= 0; i-- {
								if frame.OperandStack.size > 0 {
									topSlot := frame.OperandStack.slots[frame.OperandStack.size-1]
									frame.OperandStack.size--
									paramSlots[i] = topSlot
								}
							}
							var args []Slot
							var receiver Slot
							if opcode != OpInvokestatic {
								receiver = frame.OperandStack.PopSlot()
							}
							callerMethodName := ""
							if cfFrame.method != nil {
								callerMethodName = cf.ConstantPool.GetUtf8(cfFrame.method.NameIndex)
							}
							adjustKnightOnlinePotionFallback(
								cf.GetClassName(),
								callerMethodName,
								clsName,
								mName,
								mDesc,
								paramSlots,
							)
							if ctx.debugAppearanceSynced && ctx.debugPlayerSpriteCalls < 80 &&
								cf.GetClassName() == "fa" && clsName == "ak" && mName == "d" && mDesc == "(II)Lff;" && len(paramSlots) == 2 {
								cnG, _ := ctx.GetStaticField("cn.g").Ref.(*Object)
								if frame.LocalVars.GetRef(0) == cnG {
									ctx.debugPlayerSpriteCalls++
									log.Printf("[DEBUG-PLAYER-SPRITE] caller=fa.%s layer=%d sprite=%d direction=%d frame=%d",
										callerMethodName, paramSlots[0].Num, paramSlots[1].Num, cnG.GetField("cH").Num, cnG.GetField("cf").Num)
								}
							}
							if opcode != OpInvokestatic {
								args = append(args, receiver)
							}
							args = append(args, paramSlots...)

							if clsName == "javax/microedition/lcdui/Graphics" {
								if len(args) > 0 {
									if ctx.handleGraphicsCall(mName, mDesc, args[0].Ref, args[1:]) {
										continue
									}
								}
							}

							retSlot, err := ctx.InvokeMethod(targetCf, method, args)
							if err != nil {
								if jerr, ok := err.(*JavaException); ok {
									frame.PendingException = jerr.Object
									if ctx.checkException(cf, cfFrame.method, frame, instrPC) {
										continue
									}
									return Slot{}, err
								}
							}
							if hasReturnValue(mDesc) {
								frame.OperandStack.PushInt(retSlot.Num)
								if retSlot.Ref != nil {
									frame.OperandStack.slots[frame.OperandStack.size-1].Ref = retSlot.Ref
								}
								endIdx := strings.Index(mDesc, ")")
								if endIdx != -1 && endIdx+1 < len(mDesc) {
									retType := mDesc[endIdx+1]
									if retType == 'J' || retType == 'D' {
										frame.OperandStack.PushInt(0)
									}
								}
							}
						} else {
							if mName == "toString" || mName == "equals" || mName == "hashCode" || mName == "getClass" {
								ctx.handleNativeCall("java/lang/Object", mName, mDesc, frame, opcode)
							} else {
								paramCount := countMethodParamSlots(mDesc)
								for i := 0; i < paramCount; i++ {
									frame.OperandStack.PopRef()
								}
								if opcode != OpInvokestatic {
									frame.OperandStack.PopRef()
								}
								if hasReturnValue(mDesc) {
									frame.OperandStack.PushInt(0)
								}
							}
						}
					} else {
						if mName == "toString" || mName == "equals" || mName == "hashCode" || mName == "getClass" {
							ctx.handleNativeCall("java/lang/Object", mName, mDesc, frame, opcode)
						} else {
							paramCount := countMethodParamSlots(mDesc)
							for i := 0; i < paramCount; i++ {
								frame.OperandStack.PopRef()
							}
							if opcode != OpInvokestatic {
								frame.OperandStack.PopRef()
							}
							if hasReturnValue(mDesc) {
								frame.OperandStack.PushInt(0)
							}
						}
					}
				}
			}
		case OpAthrow:
			exRef := frame.OperandStack.PopRef()
			exObj, _ := exRef.(*Object)
			if exObj == nil {
				exObj = ctx.NewException("java/lang/NullPointerException", "athrow null")
			}
			frame.PendingException = exObj
		default:
			startPC := frame.PC - 1
			length := OpcodeLengths[opcode]
			if length > 1 {
				frame.PC = startPC + length
			}
		}

		if frame.PendingException != nil {
			if ctx.checkException(cf, cfFrame.method, frame, instrPC) {
				continue
			}
			return Slot{}, &JavaException{Object: frame.PendingException}
		}
	}
	return Slot{}, nil
}

func (ctx *VMContext) pushConstant(cf *classfile.ClassFile, frame *Frame, idx uint16) {
	cpItem := cf.ConstantPool[idx]
	switch c := cpItem.(type) {
	case classfile.ConstantIntegerInfo:
		frame.OperandStack.PushInt(c.Value)
	case classfile.ConstantLongInfo:
		frame.OperandStack.PushInt(int32(c.Value >> 32))
		frame.OperandStack.PushInt(int32(c.Value))
	case classfile.ConstantDoubleInfo:
		bits := math.Float64bits(c.Value)
		frame.OperandStack.PushInt(int32(bits >> 32))
		frame.OperandStack.PushInt(int32(bits))
	case classfile.ConstantStringInfo:
		frame.OperandStack.PushRef(cf.ConstantPool.GetUtf8(c.StringIndex))
	case classfile.ConstantUtf8Info:
		frame.OperandStack.PushRef(c.Str)
	default:
		frame.OperandStack.PushRef(nil)
	}
}

var dummyPngData []byte

func createMultiArray(desc string, counts []int) interface{} {
	if len(counts) == 0 {
		return nil
	}
	size := counts[0]
	if size < 0 {
		size = 0
	}
	if len(counts) == 1 {
		elemType := desc
		if strings.HasPrefix(elemType, "[") {
			elemType = elemType[1:]
		}
		if strings.HasPrefix(elemType, "[") || strings.HasPrefix(elemType, "L") {
			return make([]Slot, size)
		}
		switch elemType {
		case "I":
			return make([]int32, size)
		case "B", "Z":
			return make([]byte, size)
		case "S":
			return make([]int16, size)
		case "C":
			return make([]uint16, size)
		case "J":
			return make([]int64, size)
		case "F":
			return make([]float32, size)
		case "D":
			return make([]float64, size)
		default:
			return make([]Slot, size)
		}
	}

	nextDesc := desc
	if strings.HasPrefix(nextDesc, "[") {
		nextDesc = nextDesc[1:]
	}
	slots := make([]Slot, size)
	for i := 0; i < size; i++ {
		subArr := createMultiArray(nextDesc, counts[1:])
		slots[i] = Slot{Ref: subArr}
	}
	return slots
}

func init() {
	buf := new(bytes.Buffer)
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	png.Encode(buf, img)
	dummyPngData = buf.Bytes()
}

func makeFallbackImage(w, h int) *Object {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	return &Object{Extra: img}
}

func (ctx *VMContext) getResourceBytes(name string) []byte {
	if name == "" {
		return nil
	}
	name = strings.TrimPrefix(name, "/")
	name = strings.ReplaceAll(name, "x0/", "x1/")
	name = strings.ReplaceAll(name, "xundefined/", "x1/")
	if strings.HasPrefix(name, "bg/") || strings.HasPrefix(name, "icon/") || strings.HasPrefix(name, "ui/") || strings.HasPrefix(name, "font/") {
		name = "x1/" + name
	}

	if ctx.ClassLoader == nil || ctx.ClassLoader.Archive == nil {
		return nil
	}

	if b, ok := ctx.ClassLoader.Archive.Resources[name]; ok {
		return b
	}
	if b, ok := ctx.ClassLoader.Archive.Resources["/"+name]; ok {
		return b
	}

	if name == "xpng" || name == "xpng.png" {
		name = "x1/bg/sea.png"
	}

	baseNames := []string{
		name,
		name + ".png",
		name + ".jpg",
		name + ".bmp",
		name + ".vng",
	}

	var keysToTry []string
	for _, bn := range baseNames {
		keysToTry = append(keysToTry,
			bn,
			"x1/"+bn,
			"x2/"+bn,
			"x3/"+bn,
			"x4/"+bn,
			strings.TrimPrefix(bn, "x1/"),
			strings.TrimPrefix(bn, "x2/"),
			strings.TrimPrefix(bn, "x3/"),
			strings.TrimPrefix(bn, "x4/"),
		)
	}

	for _, k := range keysToTry {
		if b, ok := ctx.ClassLoader.Archive.Resources[k]; ok {
			return b
		}
		if b, ok := ctx.ClassLoader.Archive.Classes[k]; ok {
			return b
		}
	}

	for resKey, resBytes := range ctx.ClassLoader.Archive.Resources {
		if strings.HasSuffix(resKey, "/"+name) || strings.HasSuffix(resKey, name) || strings.HasSuffix(name, resKey) {
			return resBytes
		}
	}

	return nil
}

func isGraphicsMethodName(mName string) bool {
	switch mName {
	case "drawImage", "drawRegion", "fillRect", "drawRect", "drawLine", "drawString",
		"setClip", "clipRect", "translate", "getClipX", "getClipY", "getClipWidth", "getClipHeight",
		"setColor", "getColor", "drawRGB", "fillArc", "drawArc", "fillTriangle":
		return true
	}
	return false
}

func extractGraphics(ref interface{}, defaultGfx *midp.Graphics) *midp.Graphics {
	if ref != nil {
		if g, ok := ref.(*midp.Graphics); ok && g != nil {
			return g
		}
		if obj, ok := ref.(*Object); ok && obj != nil {
			if g, ok := obj.Extra.(*midp.Graphics); ok && g != nil {
				return g
			}
			if val := obj.GetField("a"); val.Ref != nil {
				if g, ok := val.Ref.(*midp.Graphics); ok && g != nil {
					return g
				}
			}
			if val := obj.GetField("graphics"); val.Ref != nil {
				if g, ok := val.Ref.(*midp.Graphics); ok && g != nil {
					return g
				}
			}
			for _, val := range obj.GetFieldsCopy() {
				if val.Ref != nil {
					if g, ok := val.Ref.(*midp.Graphics); ok && g != nil {
						return g
					}
				}
			}
		}
	}
	return defaultGfx
}

func (ctx *VMContext) isCanvas(obj *Object) bool {
	if obj == nil || obj.Class == nil {
		return false
	}
	if obj.Class.GetMethod("pointerPressed", "(II)V") != nil ||
		obj.Class.GetMethod("paint", "(Ljavax/microedition/lcdui/Graphics;)V") != nil {
		return true
	}
	cur := obj.Class
	for cur != nil {
		cName := cur.GetClassName()
		if cName == "javax/microedition/lcdui/Canvas" {
			return true
		}
		superName := cur.GetSuperClassName()
		if superName == "" || superName == "java/lang/Object" {
			break
		}
		if ctx.ClassLoader == nil {
			break
		}
		next, err := ctx.ClassLoader.LoadClass(superName)
		if err != nil || next == nil {
			break
		}
		cur = next
	}
	return false
}

func (ctx *VMContext) HandleNativeCall(clsName, mName, mDesc string, frame *Frame, opcode byte) {
	ctx.handleNativeCall(clsName, mName, mDesc, frame, opcode)
}

func (ctx *VMContext) handleNativeCall(clsName, mName, mDesc string, frame *Frame, opcode byte) {
	if clsName == "java/util/Random" && ctx.handleRandomCall(mName, mDesc, frame) {
		return
	}
	if clsName == "java/util/Hashtable" && ctx.handleHashtableNative(mName, mDesc, frame, opcode) {
		return
	}
	if clsName == "java/util/Enumeration" && ctx.handleEnumerationNative(mName, mDesc, frame, opcode) {
		return
	}
	if IsIdentityXOR(clsName, mName, mDesc) {
		// Identity stub: return the last integer argument unchanged so SocketTap owns the cipher.
		v := frame.OperandStack.PopInt()
		leftover := countMethodParamSlots(mDesc) - 1
		for i := 0; i < leftover; i++ {
			frame.OperandStack.PopSlot()
		}
		if opcode != OpInvokestatic {
			frame.OperandStack.PopRef()
		}
		if hasReturnValue(mDesc) {
			frame.OperandStack.PushInt(v)
		}
		return
	}
	if clsName == "java/lang/Thread" {
		if mName == "<init>" {
			paramCount := countMethodParamSlots(mDesc)
			var paramSlots []Slot
			for i := 0; i < paramCount; i++ {
				paramSlots = append(paramSlots, Slot{Ref: frame.OperandStack.PopRef()})
			}
			objRef := frame.OperandStack.PopRef()
			if obj, ok := objRef.(*Object); ok {
				for i := len(paramSlots) - 1; i >= 0; i-- {
					if rObj, ok := paramSlots[i].Ref.(*Object); ok && rObj != nil {
						obj.Extra = rObj
						break
					}
				}
			}
			return
		} else if mName == "start" {
			objRef := frame.OperandStack.PopRef()
			if obj, ok := objRef.(*Object); ok {
				var runnableObj *Object
				if rObj, ok := obj.Extra.(*Object); ok {
					runnableObj = rObj
				} else {
					runnableObj = obj
				}
				if runnableObj != nil && runnableObj.Class != nil {
					runMethod := runnableObj.Class.GetMethod("run", "()V")
					if runMethod != nil {
						log.Printf("[THREAD-START] Launching background Thread.run() on %s", runnableObj.Class.GetClassName())
						go func(targetCf *classfile.ClassFile, method *classfile.MethodInfo, targetObj *Object) {
							defer func() {
								if r := recover(); r != nil {
									log.Printf("[THREAD-CRASH] Thread.run() on %s crashed: %v", targetCf.GetClassName(), r)
								}
							}()
							// Wait until startApp/clinit releases execMu, then run
							// concurrently — do not hold execMu for the thread lifetime
							// or the infinite game loop starves the resource loader.
							ctx.execMu.Lock()
							ctx.execMu.Unlock()
							ctx.InvokeMethod(targetCf, method, []Slot{{Ref: targetObj}})
						}(runnableObj.Class, runMethod, runnableObj)
					}
				}
			}
			return
		} else if mName == "sleep" {
			low := frame.OperandStack.PopInt()
			var high int32
			if strings.Contains(mDesc, "J") {
				high = frame.OperandStack.PopInt()
			}
			ms := (int64(high) << 32) | int64(uint32(low))
			time.Sleep(time.Duration(ms) * time.Millisecond)
			return
		}
	} else if clsName == "java/lang/String" {
		if mName == "<init>" {
			if strings.Contains(mDesc, "[B") {
				if strings.Contains(mDesc, "Ljava/lang/String;") {
					_ = frame.OperandStack.PopRef() // charset
				}
				var length, offset int32
				if strings.Contains(mDesc, "II") {
					length = frame.OperandStack.PopInt()
					offset = frame.OperandStack.PopInt()
				}
				dataRef := frame.OperandStack.PopRef()
				objRef := frame.OperandStack.PopRef()
				b := extractBytes(dataRef)
				if b == nil {
					b = []byte{}
				}
				if offset < 0 {
					offset = 0
				}
				if length <= 0 || int(offset)+int(length) > len(b) {
					length = int32(len(b)) - offset
				}
				if length < 0 {
					length = 0
				}
				s := string(b[offset : offset+length])
				if obj, ok := objRef.(*Object); ok && obj != nil {
					obj.Extra = s
				}
			} else if countMethodParamSlots(mDesc) > 0 {
				src := extractString(frame.OperandStack.PopRef())
				objRef := frame.OperandStack.PopRef()
				if obj, ok := objRef.(*Object); ok && obj != nil {
					obj.Extra = src
				}
			} else {
				_ = frame.OperandStack.PopRef()
			}
			return
		} else if mName == "equals" {
			otherRef := frame.OperandStack.PopRef()
			strRef := frame.OperandStack.PopRef()
			targetStr := extractString(strRef)
			otherStr := extractString(otherRef)
			if strRef != nil && otherRef != nil && targetStr == otherStr {
				frame.OperandStack.PushInt(1)
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "compareTo" {
			otherRef := frame.OperandStack.PopRef()
			strRef := frame.OperandStack.PopRef()
			a := extractString(strRef)
			b := extractString(otherRef)
			frame.OperandStack.PushInt(int32(strings.Compare(a, b)))
			return
		} else if mName == "lastIndexOf" {
			idx := int32(-1)
			if mDesc == "(I)I" {
				ch := frame.OperandStack.PopInt()
				s := extractString(frame.OperandStack.PopRef())
				idx = int32(strings.LastIndexByte(s, byte(ch)))
			} else if mDesc == "(II)I" {
				from := frame.OperandStack.PopInt()
				ch := frame.OperandStack.PopInt()
				s := extractString(frame.OperandStack.PopRef())
				if from >= int32(len(s)) {
					from = int32(len(s) - 1)
				}
				if from >= 0 {
					idx = int32(strings.LastIndexByte(s[:from+1], byte(ch)))
				}
			} else if mDesc == "(Ljava/lang/String;)I" {
				sub := extractString(frame.OperandStack.PopRef())
				s := extractString(frame.OperandStack.PopRef())
				idx = int32(strings.LastIndex(s, sub))
			} else {
				from := frame.OperandStack.PopInt()
				sub := extractString(frame.OperandStack.PopRef())
				s := extractString(frame.OperandStack.PopRef())
				if from >= int32(len(s)) {
					from = int32(len(s) - 1)
				}
				if from >= 0 {
					idx = int32(strings.LastIndex(s[:from+1], sub))
				}
			}
			frame.OperandStack.PushInt(idx)
			return
		} else if mName == "valueOf" {
			if strings.HasPrefix(mDesc, "(I") || strings.HasPrefix(mDesc, "(B") || strings.HasPrefix(mDesc, "(S") {
				val := frame.OperandStack.PopInt()
				frame.OperandStack.PushRef(strconv.Itoa(int(val)))
				return
			} else if strings.HasPrefix(mDesc, "(Z") {
				val := frame.OperandStack.PopInt()
				if val != 0 {
					frame.OperandStack.PushRef("true")
				} else {
					frame.OperandStack.PushRef("false")
				}
				return
			} else if strings.HasPrefix(mDesc, "(C") {
				val := frame.OperandStack.PopInt()
				frame.OperandStack.PushRef(string(rune(val)))
				return
			} else if strings.HasPrefix(mDesc, "(J") {
				low := frame.OperandStack.PopInt()
				high := frame.OperandStack.PopInt()
				v := (int64(high) << 32) | int64(uint32(low))
				frame.OperandStack.PushRef(strconv.FormatInt(v, 10))
				return
			}
			valRef := frame.OperandStack.PopRef()
			if valRef == nil {
				frame.OperandStack.PushRef("null")
			} else if s, ok := valRef.(string); ok {
				frame.OperandStack.PushRef(s)
			} else if s := extractString(valRef); s != "" {
				frame.OperandStack.PushRef(s)
			} else if obj, ok := valRef.(*Object); ok && obj != nil {
				if s, ok := obj.Extra.(string); ok {
					frame.OperandStack.PushRef(s)
				} else {
					frame.OperandStack.PushRef(fmt.Sprintf("%v", obj))
				}
			} else {
				frame.OperandStack.PushRef(fmt.Sprintf("%v", valRef))
			}
			return
		} else if mName == "indexOf" {
			if mDesc == "(Ljava/lang/String;I)I" {
				fromIdx := frame.OperandStack.PopInt()
				subRef := frame.OperandStack.PopRef()
				strRef := frame.OperandStack.PopRef()
				frame.OperandStack.PushInt(javaIndexOfString(extractString(strRef), extractString(subRef), fromIdx))
				return
			} else if mDesc == "(II)I" {
				fromIdx := frame.OperandStack.PopInt()
				ch := frame.OperandStack.PopInt()
				strRef := frame.OperandStack.PopRef()
				frame.OperandStack.PushInt(javaIndexOfRune(extractString(strRef), rune(ch), fromIdx))
				return
			} else if mDesc == "(Ljava/lang/String;)I" {
				subRef := frame.OperandStack.PopRef()
				strRef := frame.OperandStack.PopRef()
				frame.OperandStack.PushInt(javaIndexOfString(extractString(strRef), extractString(subRef), 0))
				return
			} else {
				ch := frame.OperandStack.PopInt()
				strRef := frame.OperandStack.PopRef()
				frame.OperandStack.PushInt(javaIndexOfRune(extractString(strRef), rune(ch), 0))
				return
			}
		} else if mName == "getBytes" {
			strRef := frame.OperandStack.PopRef()
			strVal := extractString(strRef)
			frame.OperandStack.PushRef([]byte(strVal))
			return
		} else if mName == "toCharArray" {
			strRef := frame.OperandStack.PopRef()
			strVal := extractString(strRef)
			runes := []rune(strVal)
			slots := make([]Slot, len(runes))
			for i, r := range runes {
				slots[i] = Slot{Num: int32(r)}
			}
			frame.OperandStack.PushRef(slots)
			return
		} else if mName == "charAt" {
			idx := frame.OperandStack.PopInt()
			strRef := frame.OperandStack.PopRef()
			strVal := extractString(strRef)
			runes := []rune(strVal)
			if idx >= 0 && int(idx) < len(runes) {
				frame.OperandStack.PushInt(int32(runes[idx]))
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "length" {
			runes := []rune(extractString(frame.OperandStack.PopRef()))
			frame.OperandStack.PushInt(int32(len(runes)))
			return
		} else if mName == "substring" {
			var startIdx, endIdx int32
			if strings.Contains(mDesc, "II") {
				endIdx = frame.OperandStack.PopInt()
				startIdx = frame.OperandStack.PopInt()
			} else {
				startIdx = frame.OperandStack.PopInt()
				endIdx = -1
			}
			runes := []rune(extractString(frame.OperandStack.PopRef()))
			if endIdx < 0 {
				endIdx = int32(len(runes))
			}
			if startIdx >= 0 && int(endIdx) <= len(runes) && startIdx <= endIdx {
				frame.OperandStack.PushRef(string(runes[startIdx:endIdx]))
			} else {
				frame.OperandStack.PushRef("")
			}
			return
		} else if mName == "trim" {
			objRef := frame.OperandStack.PopRef()
			s := extractString(objRef)
			frame.OperandStack.PushRef(strings.TrimSpace(s))
			return
		} else if mName == "toLowerCase" {
			objRef := frame.OperandStack.PopRef()
			s := extractString(objRef)
			frame.OperandStack.PushRef(strings.ToLower(s))
			return
		} else if mName == "toUpperCase" {
			objRef := frame.OperandStack.PopRef()
			s := extractString(objRef)
			frame.OperandStack.PushRef(strings.ToUpper(s))
			return
		} else if mName == "startsWith" {
			if mDesc == "(Ljava/lang/String;I)Z" {
				from := frame.OperandStack.PopInt()
				prefix := extractString(frame.OperandStack.PopRef())
				s := extractString(frame.OperandStack.PopRef())
				if from >= 0 && int(from) <= len(s) && strings.HasPrefix(s[from:], prefix) {
					frame.OperandStack.PushInt(1)
				} else {
					frame.OperandStack.PushInt(0)
				}
				return
			}
			prefix := extractString(frame.OperandStack.PopRef())
			s := extractString(frame.OperandStack.PopRef())
			if strings.HasPrefix(s, prefix) {
				frame.OperandStack.PushInt(1)
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "endsWith" {
			suffix := extractString(frame.OperandStack.PopRef())
			s := extractString(frame.OperandStack.PopRef())
			if strings.HasSuffix(s, suffix) {
				frame.OperandStack.PushInt(1)
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "intern" {
			objRef := frame.OperandStack.PopRef()
			frame.OperandStack.PushRef(extractString(objRef))
			return
		} else if mName == "toString" {
			objRef := frame.OperandStack.PopRef()
			frame.OperandStack.PushRef(extractString(objRef))
			return
		}
	} else if clsName == "java/lang/Object" {
		if mName == "toString" {
			objRef := frame.OperandStack.PopRef()
			if s, ok := objRef.(string); ok {
				frame.OperandStack.PushRef(s)
			} else if obj, ok := objRef.(*Object); ok && obj != nil {
				if s, ok := obj.Extra.(string); ok {
					frame.OperandStack.PushRef(s)
				} else if strVal := extractString(obj); strVal != "" {
					frame.OperandStack.PushRef(strVal)
				} else if obj.Class != nil {
					frame.OperandStack.PushRef(fmt.Sprintf("%s@%p", obj.Class.GetClassName(), obj))
				} else {
					frame.OperandStack.PushRef("java.lang.Object")
				}
			} else if objRef != nil {
				frame.OperandStack.PushRef(extractString(objRef))
			} else {
				frame.OperandStack.PushRef("null")
			}
			return
		} else if mName == "equals" {
			otherRef := frame.OperandStack.PopRef()
			thisRef := frame.OperandStack.PopRef()
			if thisRef == otherRef {
				frame.OperandStack.PushInt(1)
			} else {
				thisStr := extractString(thisRef)
				otherStr := extractString(otherRef)
				if thisRef != nil && otherRef != nil && thisStr != "" && thisStr == otherStr {
					frame.OperandStack.PushInt(1)
				} else {
					frame.OperandStack.PushInt(0)
				}
			}
			return
		} else if mName == "hashCode" {
			objRef := frame.OperandStack.PopRef()
			if s, ok := objRef.(string); ok {
				var h int32
				for _, r := range s {
					h = 31*h + int32(r)
				}
				frame.OperandStack.PushInt(h)
			} else if obj, ok := objRef.(*Object); ok && obj != nil {
				h := int32(0)
				for _, r := range fmt.Sprintf("%p", obj) {
					h = 31*h + int32(r)
				}
				frame.OperandStack.PushInt(h)
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "getClass" {
			objRef := frame.OperandStack.PopRef()
			frame.OperandStack.PushRef(objRef)
			return
		} else if mName == "<init>" {
			if opcode != OpInvokestatic {
				_ = frame.OperandStack.PopRef()
			}
			return
		}
	} else if clsName == "java/lang/StringBuffer" {
		if mName == "<init>" {
			paramCount := countMethodParamSlots(mDesc)
			var initStr string
			if paramCount > 0 {
				val := frame.OperandStack.PopRef()
				if strings.HasPrefix(mDesc, "(Ljava/lang/String;") {
					initStr = extractString(val)
				} else if !strings.HasPrefix(mDesc, "(I") {
					initStr = extractString(val)
				}
			}
			objRef := frame.OperandStack.PopRef()
			if obj, ok := objRef.(*Object); ok {
				obj.Extra = initStr
			}
			return
		} else if mName == "append" {
			// aq.a builds "/x" + bx.b + path. append(I) must stringify the int;
			// PopRef() drops Num and produced "/x/mfont/..." instead of "/x1/mfont/...".
			var appendStr string
			switch {
			case strings.HasPrefix(mDesc, "(I") || strings.HasPrefix(mDesc, "(B") || strings.HasPrefix(mDesc, "(S"):
				appendStr = fmt.Sprintf("%d", frame.OperandStack.PopInt())
			case strings.HasPrefix(mDesc, "(J"):
				low := uint32(frame.OperandStack.PopInt())
				high := uint32(frame.OperandStack.PopInt())
				val := (int64(high) << 32) | int64(low)
				appendStr = fmt.Sprintf("%d", val)
			case strings.HasPrefix(mDesc, "(F"):
				bits := uint32(frame.OperandStack.PopInt())
				appendStr = fmt.Sprintf("%g", math.Float32frombits(bits))
			case strings.HasPrefix(mDesc, "(D"):
				low := uint32(frame.OperandStack.PopInt())
				high := uint32(frame.OperandStack.PopInt())
				bits := (uint64(high) << 32) | uint64(low)
				appendStr = fmt.Sprintf("%g", math.Float64frombits(bits))
			case strings.HasPrefix(mDesc, "(C"):
				appendStr = string(rune(frame.OperandStack.PopInt()))
			case strings.HasPrefix(mDesc, "(Z"):
				if frame.OperandStack.PopInt() != 0 {
					appendStr = "true"
				} else {
					appendStr = "false"
				}
			case strings.HasPrefix(mDesc, "([C"):
				arrRef := frame.OperandStack.PopRef()
				if chars, ok := arrRef.([]uint16); ok {
					appendStr = string(utf16.Decode(chars))
				} else if chars, ok := arrRef.([]rune); ok {
					appendStr = string(chars)
				} else {
					appendStr = extractString(arrRef)
				}
			default:
				appendStr = extractString(frame.OperandStack.PopRef())
			}
			sbRef := frame.OperandStack.PopRef()
			if sbObj, ok := sbRef.(*Object); ok && sbObj != nil {
				if curr, ok := sbObj.Extra.(string); ok {
					sbObj.Extra = curr + appendStr
				} else {
					sbObj.Extra = appendStr
				}
				frame.OperandStack.PushRef(sbObj)
			} else {
				frame.OperandStack.PushRef(sbRef)
			}
			return
		} else if mName == "toString" {
			objRef := frame.OperandStack.PopRef()
			if obj, ok := objRef.(*Object); ok {
				str, _ := obj.Extra.(string)
				frame.OperandStack.PushRef(str)
			} else if s, ok := objRef.(string); ok {
				frame.OperandStack.PushRef(s)
			} else {
				frame.OperandStack.PushRef("")
			}
			return
		}
	} else if strings.HasPrefix(clsName, "java/lang/") && (strings.HasSuffix(clsName, "Exception") || strings.HasSuffix(clsName, "Error") || clsName == "java/lang/Throwable") {
		if mName == "<init>" {
			paramCount := countMethodParamSlots(mDesc)
			var msg string
			for i := 0; i < paramCount; i++ {
				arg := frame.OperandStack.PopRef()
				if s, ok := arg.(string); ok {
					msg = s
				}
			}
			objRef := frame.OperandStack.PopRef()
			if obj, ok := objRef.(*Object); ok && obj != nil {
				if obj.Fields == nil {
					obj.Fields = make(map[string]Slot)
				}
				obj.Fields["detailMessage"] = Slot{Ref: msg}
			}
			return
		} else if mName == "printStackTrace" {
			objRef := frame.OperandStack.PopRef()
			msg := ""
			if obj, ok := objRef.(*Object); ok && obj != nil {
				msg = extractString(obj.Fields["detailMessage"].Ref)
			}
			log.Printf("[JAVA-EXCEPTION] %s: %s", clsName, msg)
			return
		} else if mName == "getMessage" {
			objRef := frame.OperandStack.PopRef()
			msg := ""
			if obj, ok := objRef.(*Object); ok && obj != nil {
				msg = extractString(obj.Fields["detailMessage"].Ref)
			}
			frame.OperandStack.PushRef(msg)
			return
		}
	} else if clsName == "java/io/DataInputStream" || clsName == "java/io/ByteArrayInputStream" || clsName == "java/io/InputStream" {
		if mName == "<init>" {
			paramCount := countMethodParamSlots(mDesc)
			var srcBytes []byte
			var srcTap *midp.SocketTap
			if paramCount > 0 {
				arg := frame.OperandStack.PopRef()
				if tap := socketTapFromRef(arg); tap != nil {
					srcTap = tap
				} else if obj, ok := arg.(*Object); ok && obj != nil {
					if reader, ok := obj.Extra.(*DataStreamReader); ok {
						srcBytes = reader.Data[reader.Pos:]
					} else if b, ok := obj.Extra.([]byte); ok {
						srcBytes = b
					}
				} else if b, ok := arg.([]byte); ok {
					srcBytes = b
				} else if reader, ok := arg.(*DataStreamReader); ok {
					srcBytes = reader.Data[reader.Pos:]
				}
			}
			objRef := frame.OperandStack.PopRef()
			if obj, ok := objRef.(*Object); ok {
				if srcTap != nil {
					obj.Extra = srcTap
					log.Printf("[DATA-STREAM-INIT] Wrapped SocketTap")
				} else {
					obj.Extra = &DataStreamReader{Data: srcBytes, Pos: 0}
				}
			}
			return
		} else if mName == "available" {
			objRef := frame.OperandStack.PopRef()
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					frame.OperandStack.PushInt(reader.Available())
					return
				}
			}
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "read" && mDesc == "()I" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				buf := make([]byte, 1)
				n, err := tap.ReadPlain(buf)
				if err != nil || n == 0 {
					frame.OperandStack.PushInt(-1)
					return
				}
				frame.OperandStack.PushInt(int32(buf[0]))
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					if reader.Pos < len(reader.Data) {
						b := uint8(reader.Data[reader.Pos])
						reader.Pos++
						frame.OperandStack.PushInt(int32(b))
						return
					}
				}
			}
			frame.OperandStack.PushInt(-1)
			return
		} else if mName == "readByte" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				ctx.pushTapInt(frame, tap, 1, true)
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					if reader.Pos < len(reader.Data) {
						b := int8(reader.Data[reader.Pos])
						reader.Pos++
						frame.OperandStack.PushInt(int32(b))
						return
					}
				}
			}
			frame.OperandStack.PushInt(-1)
			return
		} else if mName == "readUnsignedByte" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				ctx.pushTapInt(frame, tap, 1, false)
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					if reader.Pos < len(reader.Data) {
						b := uint8(reader.Data[reader.Pos])
						reader.Pos++
						frame.OperandStack.PushInt(int32(b))
						return
					}
				}
			}
			frame.OperandStack.PushInt(-1)
			return
		} else if mName == "readUTF" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				lenBuf := make([]byte, 2)
				if n, err := readTapBytes(tap, lenBuf); n < 2 || err != nil {
					frame.PendingException = ctx.NewException("java/io/EOFException", "EOF in readUTF")
					return
				}
				utfLen := int(binary.BigEndian.Uint16(lenBuf))
				data := make([]byte, utfLen)
				n, err := readTapBytes(tap, data)
				if n < utfLen || err != nil {
					frame.PendingException = ctx.NewException("java/io/EOFException", "EOF in readUTF")
					return
				}
				frame.OperandStack.PushRef(string(data[:n]))
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					frame.OperandStack.PushRef(reader.ReadUTF())
					return
				}
			}
			frame.OperandStack.PushRef("")
			return
		} else if mName == "readShort" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				ctx.pushTapInt(frame, tap, 2, true)
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					frame.OperandStack.PushInt(int32(reader.ReadShort()))
					return
				}
			}
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "readUnsignedShort" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				ctx.pushTapInt(frame, tap, 2, false)
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					if reader.Pos+2 <= len(reader.Data) {
						val := binary.BigEndian.Uint16(reader.Data[reader.Pos : reader.Pos+2])
						reader.Pos += 2
						frame.OperandStack.PushInt(int32(val))
						return
					}
				}
			}
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "readInt" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				ctx.pushTapInt(frame, tap, 4, true)
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					frame.OperandStack.PushInt(reader.ReadInt())
					return
				}
			}
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "readLong" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				buf := make([]byte, 8)
				n, err := readTapBytes(tap, buf)
				if err != nil || n < len(buf) {
					frame.PendingException = ctx.NewException("java/io/EOFException", "EOF on socket")
					return
				}
				value := binary.BigEndian.Uint64(buf)
				frame.OperandStack.PushInt(int32(value >> 32))
				frame.OperandStack.PushInt(int32(value))
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					value := uint64(reader.ReadLong())
					frame.OperandStack.PushInt(int32(value >> 32))
					frame.OperandStack.PushInt(int32(value))
					return
				}
			}
			frame.OperandStack.PushInt(0)
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "readBoolean" {
			objRef := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(objRef); tap != nil {
				ctx.pushTapInt(frame, tap, 1, false)
				if frame.OperandStack.size > 0 && frame.OperandStack.slots[frame.OperandStack.size-1].Num != 0 {
					frame.OperandStack.slots[frame.OperandStack.size-1].Num = 1
				}
				return
			}
			if obj, ok := objRef.(*Object); ok {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					if reader.ReadNextByte() != 0 {
						frame.OperandStack.PushInt(1)
					} else {
						frame.OperandStack.PushInt(0)
					}
					return
				}
			}
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "readFully" || mName == "read" {
			var length, offset int32
			var dstRef, objRef interface{}

			if strings.Contains(mDesc, "II") {
				// (byte[] b, int off, int len)
				length = frame.OperandStack.PopInt()
				offset = frame.OperandStack.PopInt()
				dstRef = frame.OperandStack.PopRef()
				objRef = frame.OperandStack.PopRef()
			} else if strings.Contains(mDesc, "[B") {
				// (byte[] b)
				dstRef = frame.OperandStack.PopRef()
				objRef = frame.OperandStack.PopRef()
				offset = 0
			} else {
				objRef = frame.OperandStack.PopRef()
			}

			if ctx.handleSocketInputRead(objRef, dstRef, offset, length, mName, mDesc, frame) {
				return
			}
			if obj, ok := objRef.(*Object); ok && obj != nil {
				if reader, ok := obj.Extra.(*DataStreamReader); ok {
					dstBytes := extractBytes(dstRef)
					if dstBytes != nil {
						if length <= 0 || int(offset)+int(length) > len(dstBytes) {
							length = int32(len(dstBytes)) - offset
						}
						if length < 0 {
							length = 0
						}
						readN := reader.ReadBytes(dstBytes, int(offset), int(length))
						if dstObj, ok := dstRef.(*Object); ok && dstObj != nil {
							dstObj.Extra = dstBytes
						}
						if hasReturnValue(mDesc) {
							frame.OperandStack.PushInt(readN)
						}
						return
					}
				}
			}
			if hasReturnValue(mDesc) {
				frame.OperandStack.PushInt(-1)
			}
			return
		} else if mName == "close" {
			paramCount := countMethodParamSlots(mDesc)
			for i := 0; i < paramCount; i++ {
				frame.OperandStack.PopRef()
			}
			var objRef interface{}
			if opcode != OpInvokestatic {
				objRef = frame.OperandStack.PopRef()
			}
			ctx.closeSocketTap(socketTapFromRef(objRef))
			if hasReturnValue(mDesc) {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "skip" || mName == "mark" || mName == "reset" {
			paramCount := countMethodParamSlots(mDesc)
			for i := 0; i < paramCount; i++ {
				frame.OperandStack.PopRef()
			}
			if opcode != OpInvokestatic {
				frame.OperandStack.PopRef()
			}
			if hasReturnValue(mDesc) {
				frame.OperandStack.PushInt(0)
			}
			return
		}
	} else if clsName == "java/io/ByteArrayOutputStream" {
		if ctx.handleByteArrayOutputStreamNative(mName, mDesc, frame, opcode) {
			return
		}
	} else if clsName == "java/io/OutputStream" || clsName == "java/io/DataOutputStream" {
		if ctx.handleOutputStreamNative(mName, mDesc, frame, opcode) {
			return
		}
	} else if clsName == "java/util/Vector" {
		if ctx.handleVectorNative(mName, mDesc, frame, opcode) {
			return
		}
	} else if clsName == "java/io/PrintStream" {
		if ctx.handlePrintStreamNative(mName, mDesc, frame, opcode) {
			return
		}
	} else if clsName == "javax/microedition/rms/RecordStore" {
		if ctx.RecordStores == nil {
			ctx.RecordStores = make(map[string]*RecordStoreData)
		}
		if mName == "openRecordStore" {
			createIfNecessary := frame.OperandStack.PopInt()
			nameRef := frame.OperandStack.PopRef()
			name := "default"
			if s, ok := nameRef.(string); ok {
				name = s
			} else if obj, ok := nameRef.(*Object); ok {
				name = extractString(obj)
			}
			store, exists := ctx.RecordStores[name]
			if !exists {
				if base := getBaselineRMSStores()[name]; base != nil {
					store = cloneRecordStoreData(base)
					ctx.RecordStores[name] = store
					exists = true
					log.Printf("[RMS] Restored baseline RecordStore '%s' (records=%d)", name, len(store.Records))
				}
			}
			if !exists {
				if createIfNecessary == 0 {
					frame.PendingException = ctx.NewException("javax/microedition/rms/RecordStoreNotFoundException", "RecordStore not found: "+name)
					return
				}
				store = &RecordStoreData{Records: make([][]byte, 0)}
				ctx.RecordStores[name] = store
			}
			log.Printf("[RMS] Opened RecordStore '%s' (records=%d)", name, len(store.Records))
			frame.OperandStack.PushRef(&Object{Extra: store})
			return
		} else if mName == "addRecord" {
			numBytes := frame.OperandStack.PopInt()
			offset := frame.OperandStack.PopInt()
			dataRef := frame.OperandStack.PopRef()
			rsObj := frame.OperandStack.PopRef()
			recID := int32(1)
			if obj, ok := rsObj.(*Object); ok && obj != nil {
				if store, ok := obj.Extra.(*RecordStoreData); ok && store != nil {
					var b []byte
					if src, ok := dataRef.([]byte); ok && int(offset)+int(numBytes) <= len(src) {
						b = make([]byte, numBytes)
						copy(b, src[offset:offset+numBytes])
					}
					store.Records = append(store.Records, b)
					recID = int32(len(store.Records))
					log.Printf("[RMS] addRecord #%d (%d bytes)", recID, len(b))
				}
			}
			frame.OperandStack.PushInt(recID)
			return
		} else if mName == "getRecord" {
			if strings.HasPrefix(mDesc, "(I[BI)") {
				// getRecord(int recordId, byte[] buffer, int offset) -> int
				offset := frame.OperandStack.PopInt()
				bufferRef := frame.OperandStack.PopRef()
				recID := frame.OperandStack.PopInt()
				rsObj := frame.OperandStack.PopRef()
				readLen := int32(-1)
				if obj, ok := rsObj.(*Object); ok && obj != nil {
					if store, ok := obj.Extra.(*RecordStoreData); ok && store != nil {
						idx := int(recID) - 1
						if idx >= 0 && idx < len(store.Records) {
							recData := store.Records[idx]
							dstBytes := extractBytes(bufferRef)
							if dstBytes != nil && int(offset)+len(recData) <= len(dstBytes) {
								copy(dstBytes[offset:], recData)
								readLen = int32(len(recData))
								log.Printf("[RMS] getRecord #%d into buffer -> %d bytes", recID, readLen)
							}
						}
					}
				}
				frame.OperandStack.PushInt(readLen)
				return
			} else {
				// getRecord(int recordId) -> byte[]
				recID := frame.OperandStack.PopInt()
				rsObj := frame.OperandStack.PopRef()
				if obj, ok := rsObj.(*Object); ok && obj != nil {
					if store, ok := obj.Extra.(*RecordStoreData); ok && store != nil {
						idx := int(recID) - 1
						if idx >= 0 && idx < len(store.Records) {
							result := store.Records[idx]
							log.Printf("[RMS] getRecord #%d -> %d bytes", recID, len(result))
							if len(result) == 0 {
								frame.OperandStack.PushRef(nil)
							} else {
								recCopy := make([]byte, len(result))
								copy(recCopy, result)
								frame.OperandStack.PushRef(recCopy)
							}
							return
						}
					}
				}
				frame.PendingException = ctx.NewException("javax/microedition/rms/InvalidRecordIDException", "Invalid record id")
				return
			}
		} else if mName == "getNumRecords" {
			rsObj := frame.OperandStack.PopRef()
			count := int32(0)
			if obj, ok := rsObj.(*Object); ok && obj != nil {
				if store, ok := obj.Extra.(*RecordStoreData); ok && store != nil {
					count = int32(len(store.Records))
				}
			}
			frame.OperandStack.PushInt(count)
			return
		} else if mName == "setRecord" {
			numBytes := frame.OperandStack.PopInt()
			offset := frame.OperandStack.PopInt()
			dataRef := frame.OperandStack.PopRef()
			recID := frame.OperandStack.PopInt()
			rsObj := frame.OperandStack.PopRef()
			if obj, ok := rsObj.(*Object); ok && obj != nil {
				if store, ok := obj.Extra.(*RecordStoreData); ok && store != nil {
					idx := int(recID) - 1
					if idx >= 0 && idx < len(store.Records) {
						srcBytes := extractBytes(dataRef)
						if srcBytes != nil && int(offset)+int(numBytes) <= len(srcBytes) {
							b := make([]byte, numBytes)
							copy(b, srcBytes[offset:offset+numBytes])
							store.Records[idx] = b
							log.Printf("[RMS] setRecord #%d (%d bytes)", recID, len(b))
						}
					}
				}
			}
			return
		} else if mName == "enumerateRecords" {
			keepUpdated := frame.OperandStack.PopSlot()
			_ = keepUpdated
			compRef := frame.OperandStack.PopRef()
			_ = compRef
			filterRef := frame.OperandStack.PopRef()
			_ = filterRef
			rsObj := frame.OperandStack.PopRef()
			_ = rsObj
			enumObj := &Object{Extra: "RecordEnumeration"}
			frame.OperandStack.PushRef(enumObj)
			return
		} else if mName == "closeRecordStore" {
			rsObj := frame.OperandStack.PopRef()
			_ = rsObj
			return
		} else if mName == "listRecordStores" {
			if len(ctx.RecordStores) == 0 {
				frame.OperandStack.PushRef(nil)
				return
			}
			storeNames := make([]string, 0, len(ctx.RecordStores))
			for name := range ctx.RecordStores {
				storeNames = append(storeNames, name)
			}
			sort.Strings(storeNames)
			names := make([]interface{}, len(storeNames))
			for i, name := range storeNames {
				names[i] = ctx.NewString(name)
			}
			frame.OperandStack.PushRef(names)
			return
		} else if mName == "deleteRecordStore" {
			nameRef := frame.OperandStack.PopRef()
			name := extractString(nameRef)
			if _, ok := ctx.RecordStores[name]; ok {
				delete(ctx.RecordStores, name)
				log.Printf("[RMS] Deleted RecordStore '%s'", name)
			} else {
				frame.PendingException = ctx.NewException("javax/microedition/rms/RecordStoreNotFoundException", "RecordStore not found: "+name)
			}
			return
		}
	} else if clsName == "javax/microedition/rms/RecordEnumeration" {
		if mName == "hasNextElement" || mName == "hasPreviousElement" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "nextRecord" || mName == "previousRecord" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushRef(nil)
			return
		} else if mName == "nextRecordId" || mName == "numRecords" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(0)
			return
		} else if mName == "destroy" || mName == "reset" || mName == "rebuild" {
			frame.OperandStack.PopRef()
			return
		}
	} else if clsName == "javax/microedition/lcdui/Display" {
		if mName == "getDisplay" {
			frame.OperandStack.PopRef() // MIDlet
			frame.OperandStack.PushRef(ctx)
			return
		} else if mName == "setCurrent" {
			displayable := frame.OperandStack.PopRef()
			if obj, ok := displayable.(*Object); ok && obj != nil {
				if ctx.isCanvas(obj) {
					ctx.CurrentCanvas = obj
					ctx.PendingTextBox = nil
				} else {
					ctx.PendingTextBox = obj
				}
				if obj.Class != nil {
					if szM := obj.Class.GetMethod("sizeChanged", "(II)V"); szM != nil {
						log.Printf("[MIDP-EVENT] Calling sizeChanged(240, 320) on %s", obj.Class.GetClassName())
						ctx.InvokeMethod(obj.Class, szM, []Slot{{Ref: obj}, {Num: 240}, {Num: 320}})
					}
					if snM := obj.Class.GetMethod("showNotify", "()V"); snM != nil {
						log.Printf("[MIDP-EVENT] Calling showNotify() on %s", obj.Class.GetClassName())
						ctx.InvokeMethod(obj.Class, snM, []Slot{{Ref: obj}})
					}
				}
			}
			frame.OperandStack.PopRef() // Display instance
			return
		} else if mName == "isColor" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(1)
			return
		} else if mName == "numColors" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(65536)
			return
		}
	} else if clsName == "javax/microedition/lcdui/TextBox" {
		if mName == "<init>" {
			constraints := frame.OperandStack.PopInt()
			maxSize := frame.OperandStack.PopInt()
			textRef := frame.OperandStack.PopRef()
			titleRef := frame.OperandStack.PopRef()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				obj.SetField("title", Slot{Ref: titleRef})
				obj.SetField("text", Slot{Ref: textRef})
				obj.SetField("maxSize", Slot{Num: maxSize})
				obj.SetField("constraints", Slot{Num: constraints})
			}
			return
		} else if mName == "setString" {
			textRef := frame.OperandStack.PopRef()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				obj.SetField("text", Slot{Ref: textRef})
			}
			return
		} else if mName == "getString" {
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				val := obj.GetField("text").Ref
				if val == nil {
					val = ctx.NewString("")
				}
				frame.OperandStack.PushRef(val)
			} else {
				frame.OperandStack.PushRef(ctx.NewString(""))
			}
			return
		} else if mName == "setMaxSize" {
			maxSize := frame.OperandStack.PopInt()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				obj.SetField("maxSize", Slot{Num: maxSize})
			}
			frame.OperandStack.PushInt(maxSize)
			return
		} else if mName == "getMaxSize" {
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				frame.OperandStack.PushInt(obj.GetField("maxSize").Num)
			} else {
				frame.OperandStack.PushInt(200)
			}
			return
		} else if mName == "setConstraints" {
			constraints := frame.OperandStack.PopInt()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				obj.SetField("constraints", Slot{Num: constraints})
			}
			return
		} else if mName == "getConstraints" {
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				frame.OperandStack.PushInt(obj.GetField("constraints").Num)
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "size" {
			thisRef := frame.OperandStack.PopRef()
			str := ""
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				str = extractString(obj.GetField("text").Ref)
			}
			frame.OperandStack.PushInt(int32(len([]rune(str))))
			return
		} else {
			paramCount := countMethodParamSlots(mDesc)
			for i := 0; i < paramCount; i++ {
				frame.OperandStack.PopRef()
			}
			if opcode != OpInvokestatic {
				frame.OperandStack.PopRef()
			}
			if hasReturnValue(mDesc) {
				frame.OperandStack.PushInt(0)
			}
			return
		}
	} else if clsName == "javax/microedition/lcdui/Command" {
		if mName == "<init>" {
			priority := frame.OperandStack.PopInt()
			cmdType := frame.OperandStack.PopInt()
			labelRef := frame.OperandStack.PopRef()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				obj.SetField("label", Slot{Ref: labelRef})
				obj.SetField("cmdType", Slot{Num: cmdType})
				obj.SetField("priority", Slot{Num: priority})
			}
			return
		} else if mName == "getLabel" {
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				frame.OperandStack.PushRef(obj.GetField("label").Ref)
			} else {
				frame.OperandStack.PushRef(ctx.NewString(""))
			}
			return
		} else if mName == "getCommandType" {
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				frame.OperandStack.PushInt(obj.GetField("cmdType").Num)
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "getPriority" {
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				frame.OperandStack.PushInt(obj.GetField("priority").Num)
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		}
	} else if clsName == "javax/microedition/lcdui/Canvas" || clsName == "javax/microedition/lcdui/Displayable" || clsName == "javax/microedition/lcdui/Screen" {
		if mName == "getWidth" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(240)
			return
		} else if mName == "getHeight" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(320)
			return
		} else if mName == "isShown" || mName == "hasPointerEvents" || mName == "hasRepeatEvents" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(1)
			return
		} else if mName == "getTitle" {
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				titleRef := obj.GetField("title").Ref
				if titleRef == nil {
					titleRef = ctx.NewString("")
				}
				frame.OperandStack.PushRef(titleRef)
			} else {
				frame.OperandStack.PushRef(ctx.NewString(""))
			}
			return
		} else if mName == "setTitle" {
			titleRef := frame.OperandStack.PopRef()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				obj.SetField("title", Slot{Ref: titleRef})
			}
			return
		} else if mName == "repaint" {
			paramCount := countMethodParamSlots(mDesc)
			for i := 0; i < paramCount; i++ {
				frame.OperandStack.PopRef()
			}
			frame.OperandStack.PopRef() // Canvas receiver
			ctx.RepaintPending = true
			return
		} else if mName == "serviceRepaints" {
			paramCount := countMethodParamSlots(mDesc)
			for i := 0; i < paramCount; i++ {
				frame.OperandStack.PopRef()
			}
			canvasRef := frame.OperandStack.PopRef()
			if canvasObj, ok := canvasRef.(*Object); ok && canvasObj != nil && canvasObj.Class != nil {
				if ctx.Graphics != nil {
					ctx.RepaintPending = false
					ctx.Graphics.BeginPaint()
					paintM := canvasObj.Class.GetMethod("paint", "(Ljavax/microedition/lcdui/Graphics;)V")
					if paintM != nil {
						ctx.InvokeMethod(canvasObj.Class, paintM, []Slot{
							{Ref: canvasObj},
							{Ref: ctx.Graphics},
						})
					}
					ctx.Graphics.EndPaint()
				}
			}
			return
		} else if mName == "setCommandListener" {
			listenerRef := frame.OperandStack.PopRef()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				obj.SetField("listener", Slot{Ref: listenerRef})
			}
			return
		} else if mName == "addCommand" {
			cmdRef := frame.OperandStack.PopRef()
			thisRef := frame.OperandStack.PopRef()
			if obj, ok := thisRef.(*Object); ok && obj != nil {
				if cmdObj, ok := cmdRef.(*Object); ok && cmdObj != nil {
					label := extractString(cmdObj.GetField("label").Ref)
					if label == "OK" || label == "Ok" || cmdObj.GetField("cmdType").Num == 4 {
						obj.SetField("okCommand", Slot{Ref: cmdRef})
					} else if label == "Cancel" || cmdObj.GetField("cmdType").Num == 3 {
						obj.SetField("cancelCommand", Slot{Ref: cmdRef})
					}
				}
			}
			return
		}
	} else if clsName == "javax/microedition/lcdui/Graphics" || strings.HasSuffix(clsName, "/Graphics") {
		if mName == "translate" {
			dy := frame.OperandStack.PopInt()
			dx := frame.OperandStack.PopInt()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				g.Translate(int(dx), int(dy))
			}
			return
		} else if mName == "getTranslateX" {
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				frame.OperandStack.PushInt(int32(g.TransX))
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "getTranslateY" {
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				frame.OperandStack.PushInt(int32(g.TransY))
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "setClip" {
			h := frame.OperandStack.PopInt()
			w := frame.OperandStack.PopInt()
			y := frame.OperandStack.PopInt()
			x := frame.OperandStack.PopInt()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				if w <= 0 {
					w = int32(g.Width)
				}
				if h <= 0 {
					h = int32(g.Height)
				}
				g.SetClip(int(x), int(y), int(w), int(h))
			}
			return
		} else if mName == "clipRect" {
			h := frame.OperandStack.PopInt()
			w := frame.OperandStack.PopInt()
			y := frame.OperandStack.PopInt()
			x := frame.OperandStack.PopInt()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				g.ClipRect(int(x), int(y), int(w), int(h))
			}
			return
		} else if mName == "getClipX" {
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				frame.OperandStack.PushInt(int32(g.ClipX - g.TransX))
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "getClipY" {
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				frame.OperandStack.PushInt(int32(g.ClipY - g.TransY))
			} else {
				frame.OperandStack.PushInt(0)
			}
			return
		} else if mName == "getClipWidth" {
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil && g.ClipW > 0 {
				frame.OperandStack.PushInt(int32(g.ClipW))
			} else {
				frame.OperandStack.PushInt(240)
			}
			return
		} else if mName == "getClipHeight" {
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil && g.ClipH > 0 {
				frame.OperandStack.PushInt(int32(g.ClipH))
			} else {
				frame.OperandStack.PushInt(320)
			}
			return
		} else if mName == "setColor" {
			if strings.Contains(mDesc, "(I)V") {
				rgb := frame.OperandStack.PopInt()
				gfxRef := frame.OperandStack.PopRef()
				g := extractGraphics(gfxRef, ctx.Graphics)
				r := (rgb >> 16) & 0xFF
				gVal := (rgb >> 8) & 0xFF
				b := rgb & 0xFF
				if g != nil {
					g.SetColor(int(r), int(gVal), int(b))
				}
			} else {
				b := frame.OperandStack.PopInt()
				gVal := frame.OperandStack.PopInt()
				r := frame.OperandStack.PopInt()
				gfxRef := frame.OperandStack.PopRef()
				g := extractGraphics(gfxRef, ctx.Graphics)
				if g != nil {
					g.SetColor(int(r), int(gVal), int(b))
				}
			}
			return
		} else if mName == "fillRect" {
			h := frame.OperandStack.PopInt()
			w := frame.OperandStack.PopInt()
			y := frame.OperandStack.PopInt()
			x := frame.OperandStack.PopInt()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				g.FillRect(int(x), int(y), int(w), int(h))
			}
			return
		} else if mName == "drawRect" {
			h := frame.OperandStack.PopInt()
			w := frame.OperandStack.PopInt()
			y := frame.OperandStack.PopInt()
			x := frame.OperandStack.PopInt()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				g.DrawRect(int(x), int(y), int(w), int(h))
			}
			return
		} else if mName == "drawLine" {
			y2 := frame.OperandStack.PopInt()
			x2 := frame.OperandStack.PopInt()
			y1 := frame.OperandStack.PopInt()
			x1 := frame.OperandStack.PopInt()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			if g != nil {
				g.DrawLine(int(x1), int(y1), int(x2), int(y2))
			}
			return
		} else if mName == "drawString" {
			anchor := frame.OperandStack.PopInt()
			y := frame.OperandStack.PopInt()
			x := frame.OperandStack.PopInt()
			strRef := frame.OperandStack.PopRef()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			strVal := extractString(strRef)
			if g != nil && len(strVal) > 0 {
				g.DrawString(strVal, int(x), int(y), int(anchor))
			}
			return
		} else if mName == "drawImage" {
			anchor := frame.OperandStack.PopInt()
			y := frame.OperandStack.PopInt()
			x := frame.OperandStack.PopInt()
			imgRef := frame.OperandStack.PopRef()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			realImg := extractImage(imgRef)
			if realImg != nil && g != nil {
				g.DrawImage(realImg, int(x), int(y), int(anchor))
			}
			return
		} else if mName == "drawRegion" {
			anchor := frame.OperandStack.PopInt()
			yDest := frame.OperandStack.PopInt()
			xDest := frame.OperandStack.PopInt()
			transform := frame.OperandStack.PopInt()
			h := frame.OperandStack.PopInt()
			w := frame.OperandStack.PopInt()
			ySrc := frame.OperandStack.PopInt()
			xSrc := frame.OperandStack.PopInt()
			imgRef := frame.OperandStack.PopRef()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			realImg := extractImage(imgRef)
			if realImg != nil && g != nil {
				g.DrawRegion(realImg, int(xSrc), int(ySrc), int(w), int(h), int(transform), int(xDest), int(yDest), int(anchor))
			}
			return
		} else if mName == "drawRGB" {
			processAlpha := frame.OperandStack.PopInt()
			h := frame.OperandStack.PopInt()
			w := frame.OperandStack.PopInt()
			y := frame.OperandStack.PopInt()
			x := frame.OperandStack.PopInt()
			scanlength := frame.OperandStack.PopInt()
			offset := frame.OperandStack.PopInt()
			rgbRef := frame.OperandStack.PopRef()
			gfxRef := frame.OperandStack.PopRef()
			g := extractGraphics(gfxRef, ctx.Graphics)
			var rgbArray []int32
			if obj, ok := rgbRef.(*Object); ok && obj != nil {
				if arr, ok := obj.Extra.([]int32); ok {
					rgbArray = arr
				}
			} else if arr, ok := rgbRef.([]int32); ok {
				rgbArray = arr
			}
			if g != nil {
				g.DrawRGB(rgbArray, int(offset), int(scanlength), int(x), int(y), int(w), int(h), processAlpha != 0)
			}
			return
		}
	} else if clsName == "javax/microedition/lcdui/Font" {
		if mName == "getFont" {
			_ = frame.OperandStack.PopInt()
			_ = frame.OperandStack.PopInt()
			_ = frame.OperandStack.PopInt()
			fontObj := &Object{Class: &classfile.ClassFile{ConstantPool: make(classfile.ConstantPool, 10)}}
			frame.OperandStack.PushRef(fontObj)
			return
		} else if mName == "getDefaultFont" {
			fontObj := &Object{Class: &classfile.ClassFile{ConstantPool: make(classfile.ConstantPool, 10)}}
			frame.OperandStack.PushRef(fontObj)
			return
		} else if mName == "stringWidth" {
			strRef := frame.OperandStack.PopRef()
			frame.OperandStack.PopRef()
			strVal := extractString(strRef)
			w := int32(len([]rune(strVal)) * 8)
			frame.OperandStack.PushInt(w)
			return
		} else if mName == "substringWidth" {
			length := frame.OperandStack.PopInt()
			_ = frame.OperandStack.PopInt()
			_ = frame.OperandStack.PopRef()
			frame.OperandStack.PopRef()
			w := length * 8
			frame.OperandStack.PushInt(w)
			return
		} else if mName == "charWidth" {
			_ = frame.OperandStack.PopInt()
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(8)
			return
		} else if mName == "getHeight" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(13)
			return
		} else if mName == "getBaselinePosition" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(10)
			return
		} else if mName == "getStyle" || mName == "getSize" || mName == "getFace" {
			frame.OperandStack.PopRef()
			frame.OperandStack.PushInt(0)
			return
		}
	} else if clsName == "javax/microedition/lcdui/Image" {
		if mName == "createImage" {
			log.Printf("[CREATE-IMAGE-ENTRY] mDesc=%s", mDesc)
			if strings.Contains(mDesc, "(II)") {
				// createImage(int width, int height)
				h := frame.OperandStack.PopInt()
				w := frame.OperandStack.PopInt()
				if w <= 0 {
					w = 1
				}
				if h <= 0 {
					h = 1
				}
				rgba := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
				gfx := midp.NewGraphics(int(w), int(h))
				gfx.Buffer = rgba
				log.Printf("[IMAGE-OFFSCREEN] Created offscreen mutable image (%dx%d)", w, h)
				obj := &Object{Extra: rgba}
				obj.SetField("graphics", Slot{Ref: gfx})
				frame.OperandStack.PushRef(obj)
				return
			} else if strings.Contains(mDesc, "InputStream") {
				streamRef := frame.OperandStack.PopRef()
				var streamBytes []byte
				if obj, ok := streamRef.(*Object); ok && obj != nil {
					if reader, ok := obj.Extra.(*DataStreamReader); ok {
						streamBytes = reader.Data[reader.Pos:]
					} else if b, ok := obj.Extra.([]byte); ok {
						streamBytes = b
					}
				} else if b, ok := streamRef.([]byte); ok {
					streamBytes = b
				}
				if streamBytes != nil {
					img, err := midp.DecodeImage(streamBytes)
					if err == nil && img != nil {
						log.Printf("[IMAGE-STREAM-SUCCESS] Decoded InputStream image (%dx%d)", img.Bounds().Dx(), img.Bounds().Dy())
						frame.OperandStack.PushRef(&Object{Extra: img})
						return
					} else {
						log.Printf("[IMAGE-STREAM-FAIL] Decode failed: %v (len=%d)", err, len(streamBytes))
					}
				}
				frame.PendingException = ctx.NewException("java/io/IOException", "Image stream decode failed")
				return
			} else if strings.Contains(mDesc, "([BII)") || (strings.Contains(mDesc, "[B") && countMethodParamSlots(mDesc) >= 3) {
				// createImage(byte[] data, int offset, int length)
				length := frame.OperandStack.PopInt()
				offset := frame.OperandStack.PopInt()
				dataRef := frame.OperandStack.PopRef()
				dataBytes := extractBytes(dataRef)
				if dataBytes != nil && len(dataBytes) > 0 {
					if offset < 0 || int(offset) >= len(dataBytes) {
						offset = 0
					}
					if length <= 0 || int(offset)+int(length) > len(dataBytes) {
						length = int32(len(dataBytes)) - offset
					}
					if length > 0 && int(offset)+int(length) <= len(dataBytes) {
						subData := dataBytes[offset : offset+length]
						img, err := midp.DecodeImage(subData)
						if err == nil && img != nil {
							log.Printf("[IMAGE-SUCCESS] Decoded byte array image (%dx%d)", img.Bounds().Dx(), img.Bounds().Dy())
							frame.OperandStack.PushRef(&Object{Extra: img})
							return
						} else {
							hLen := 16
							if len(subData) < 16 {
								hLen = len(subData)
							}
							log.Printf("[IMAGE-DECODE-FAIL] offset=%d length=%d dataBytesLen=%d subDataLen=%d header=%X err=%v", offset, length, len(dataBytes), len(subData), subData[:hLen], err)
						}
					}
				}
				log.Printf("[IMAGE-BYTES-FALLBACK] Returning fallback image for byte array image (dataBytes len=%d)", len(dataBytes))
				frame.OperandStack.PushRef(makeFallbackImage(120, 120))
				return
			} else if strings.Contains(mDesc, "([B)") || strings.Contains(mDesc, "[B") {
				// createImage(byte[] data)
				dataRef := frame.OperandStack.PopRef()
				dataBytes := extractBytes(dataRef)
				if dataBytes != nil && len(dataBytes) > 0 {
					img, err := midp.DecodeImage(dataBytes)
					if err == nil && img != nil {
						log.Printf("[IMAGE-SUCCESS-1PARAM] Decoded 1-param byte image (%dx%d)", img.Bounds().Dx(), img.Bounds().Dy())
						frame.OperandStack.PushRef(&Object{Extra: img})
						return
					} else {
						log.Printf("[IMAGE-DECODE-FAIL-1PARAM] Failed 1-param decode: %v (len=%d)", err, len(dataBytes))
					}
				}
				frame.OperandStack.PushRef(makeFallbackImage(120, 120))
				return
			} else if strings.Contains(mDesc, "Image;") && countMethodParamSlots(mDesc) >= 6 {
				// createImage(Image image, int x, int y, int width, int height, int transform)
				transform := frame.OperandStack.PopInt()
				h := frame.OperandStack.PopInt()
				w := frame.OperandStack.PopInt()
				y := frame.OperandStack.PopInt()
				x := frame.OperandStack.PopInt()
				srcRef := frame.OperandStack.PopRef()
				if srcImg := extractImage(srcRef); srcImg != nil && w > 0 && h > 0 {
					transImg := transformImageRegion(srcImg, int(x), int(y), int(w), int(h), int(transform))
					frame.OperandStack.PushRef(&Object{Extra: transImg})
					return
				}
				frame.OperandStack.PushRef(makeFallbackImage(int(w), int(h)))
				return
			} else {
				// createImage(String name)
				resRef := frame.OperandStack.PopRef()
				resName := extractString(resRef)
				if resName != "" {
					imgBytes := ctx.getResourceBytes(resName)
					if imgBytes != nil {
						img, err := midp.DecodeImage(imgBytes)
						if err == nil && img != nil {
							log.Printf("[IMAGE-SUCCESS] Loaded image %s (%dx%d)", resName, img.Bounds().Dx(), img.Bounds().Dy())
							frame.OperandStack.PushRef(&Object{Extra: img})
							return
						} else {
							log.Printf("[IMAGE-DECODE-FAIL] Failed to decode image %s: %v", resName, err)
						}
					} else {
						log.Printf("[IMAGE-NOT-FOUND] Resource %s not found in JAR", resName)
					}
				}
				frame.PendingException = ctx.NewException("java/io/IOException", "Resource not found: "+resName)
				return
			}
		} else if mName == "createRGBImage" {
			processAlpha := frame.OperandStack.PopInt()
			h := frame.OperandStack.PopInt()
			w := frame.OperandStack.PopInt()
			rgbRef := frame.OperandStack.PopRef()
			if w <= 0 {
				w = 1
			}
			if h <= 0 {
				h = 1
			}
			rgba := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
			if ints, ok := rgbRef.([]int32); ok {
				idx := 0
				for y := 0; y < int(h); y++ {
					for x := 0; x < int(w); x++ {
						if idx < len(ints) {
							c := ints[idx]
							idx++
							rgba.Set(x, y, color.RGBA{
								R: uint8((c >> 16) & 0xFF),
								G: uint8((c >> 8) & 0xFF),
								B: uint8(c & 0xFF),
								A: rgbAlpha(c, processAlpha != 0),
							})
						}
					}
				}
			}
			log.Printf("[IMAGE-RGB] Created RGB image (%dx%d)", w, h)
			frame.OperandStack.PushRef(&Object{Extra: rgba})
			return
		} else if mName == "getGraphics" {
			imgRef := frame.OperandStack.PopRef()
			if obj, ok := imgRef.(*Object); ok && obj != nil {
				if gfx, ok := obj.GetField("graphics").Ref.(*midp.Graphics); ok && gfx != nil {
					frame.OperandStack.PushRef(gfx)
					return
				}
				if realImg, ok := obj.Extra.(*image.RGBA); ok && realImg != nil {
					gfx := midp.NewGraphics(realImg.Bounds().Dx(), realImg.Bounds().Dy())
					gfx.Buffer = realImg
					obj.SetField("graphics", Slot{Ref: gfx})
					frame.OperandStack.PushRef(gfx)
					return
				}
			}
			frame.OperandStack.PushRef(ctx.Graphics)
			return
		} else if mName == "getWidth" {
			imgRef := frame.OperandStack.PopRef()
			if obj, ok := imgRef.(*Object); ok {
				if realImg, ok := obj.Extra.(image.Image); ok {
					frame.OperandStack.PushInt(int32(realImg.Bounds().Dx()))
					return
				}
			}
			frame.OperandStack.PushInt(10)
			return
		} else if mName == "getHeight" {
			imgRef := frame.OperandStack.PopRef()
			if obj, ok := imgRef.(*Object); ok {
				if realImg, ok := obj.Extra.(image.Image); ok {
					frame.OperandStack.PushInt(int32(realImg.Bounds().Dy()))
					return
				}
			}
			frame.OperandStack.PushInt(10)
			return
		}
	} else if clsName == "java/lang/System" {
		if mName == "currentTimeMillis" {
			now := time.Now().UnixMilli()
			frame.OperandStack.PushInt(int32(now >> 32))
			frame.OperandStack.PushInt(int32(now))
			return
		} else if mName == "gc" {
			return
		}
	} else if mName == "getResourceAsStream" {
		resRef := frame.OperandStack.PopRef()
		if opcode != OpInvokestatic {
			frame.OperandStack.PopRef()
		}
		resName := extractString(resRef)
		log.Printf("[RESOURCE-LOOKUP] resName=%s", resName)
		if resName != "" {
			bytes := ctx.getResourceBytes(resName)
			if bytes != nil {
				log.Printf("[RESOURCE-SUCCESS] Loaded resource %s (%d bytes)", resName, len(bytes))
				frame.OperandStack.PushRef(&Object{Extra: &DataStreamReader{Data: bytes, Pos: 0}})
				return
			}
			log.Printf("[RESOURCE-FAIL] Resource not found: %s", resName)
		}
		frame.OperandStack.PushRef(nil)
		return
	} else if clsName == "javax/microedition/io/Connector" {
		if mName == "open" {
			ctx.handleConnectorOpen(mDesc, frame)
			return
		}
	} else if ctx.handleHTTPNative(clsName, mName, mDesc, frame, opcode) {
		return
	} else if isSocketConnectionClass(clsName) {
		if ctx.handleSocketConnectionNative(mName, mDesc, frame, opcode) {
			return
		}
	}

	if mName == "toString" && (mDesc == "()Ljava/lang/String;" || mDesc == "") {
		var objRef interface{}
		if opcode != OpInvokestatic {
			objRef = frame.OperandStack.PopRef()
		}
		if s, ok := objRef.(string); ok {
			frame.OperandStack.PushRef(s)
		} else if obj, ok := objRef.(*Object); ok && obj != nil {
			if s, ok := obj.Extra.(string); ok {
				frame.OperandStack.PushRef(s)
			} else if strVal := extractString(obj); strVal != "" {
				frame.OperandStack.PushRef(strVal)
			} else if obj.Class != nil {
				frame.OperandStack.PushRef(fmt.Sprintf("%s@%p", obj.Class.GetClassName(), obj))
			} else {
				frame.OperandStack.PushRef("java.lang.Object")
			}
		} else if objRef != nil {
			frame.OperandStack.PushRef(extractString(objRef))
		} else {
			frame.OperandStack.PushRef("null")
		}
		return
	}

	// Generic native fallback: pop parameters and push default return slot if needed
	paramCount := countMethodParamSlots(mDesc)
	for i := 0; i < paramCount; i++ {
		frame.OperandStack.PopRef()
	}
	if opcode != OpInvokestatic {
		frame.OperandStack.PopRef()
	}
	if hasReturnValue(mDesc) {
		frame.OperandStack.PushInt(0)
	}
}

func (ctx *VMContext) handlePrintStreamNative(mName, mDesc string, frame *Frame, opcode byte) bool {
	switch mName {
	case "println", "print":
		var val interface{} = ""
		if mDesc == "()V" {
			val = ""
		} else if strings.HasPrefix(mDesc, "(Ljava/lang/String;") || strings.HasPrefix(mDesc, "(Ljava/lang/Object;") {
			valRef := frame.OperandStack.PopRef()
			if valRef == nil {
				val = "null"
			} else {
				val = extractString(valRef)
			}
		} else if strings.HasPrefix(mDesc, "(I") || strings.HasPrefix(mDesc, "(S") || strings.HasPrefix(mDesc, "(B") {
			val = frame.OperandStack.PopInt()
		} else if strings.HasPrefix(mDesc, "(C") {
			val = string(rune(frame.OperandStack.PopInt()))
		} else if strings.HasPrefix(mDesc, "(Z") {
			if frame.OperandStack.PopInt() != 0 {
				val = "true"
			} else {
				val = "false"
			}
		} else if strings.HasPrefix(mDesc, "(J") {
			low := frame.OperandStack.PopSlot()
			high := frame.OperandStack.PopSlot()
			val = (int64(high.Num) << 32) | (int64(uint32(low.Num)))
		} else if strings.HasPrefix(mDesc, "(D") {
			frame.OperandStack.PopSlot()
			frame.OperandStack.PopSlot()
			val = 0.0
		} else if strings.HasPrefix(mDesc, "(F") {
			val = frame.OperandStack.PopInt()
		} else if strings.HasPrefix(mDesc, "([C") {
			arrRef := frame.OperandStack.PopRef()
			if chars, ok := arrRef.([]uint16); ok {
				runes := make([]rune, len(chars))
				for i, c := range chars {
					runes[i] = rune(c)
				}
				val = string(runes)
			} else if slots, ok := arrRef.([]Slot); ok {
				runes := make([]rune, len(slots))
				for i, s := range slots {
					runes[i] = rune(s.Num)
				}
				val = string(runes)
			} else {
				val = ""
			}
		} else if strings.HasPrefix(mDesc, "([B") {
			arrRef := frame.OperandStack.PopRef()
			val = string(extractBytes(arrRef))
		} else {
			paramCount := countMethodParamSlots(mDesc)
			for i := 0; i < paramCount; i++ {
				frame.OperandStack.PopSlot()
			}
		}
		if opcode != OpInvokestatic {
			_ = frame.OperandStack.PopRef()
		}
		log.Printf("[JAVA-OUT] %v", val)
		return true
	case "write":
		if strings.Contains(mDesc, "II") {
			length := frame.OperandStack.PopInt()
			offset := frame.OperandStack.PopInt()
			srcRef := frame.OperandStack.PopRef()
			if opcode != OpInvokestatic {
				_ = frame.OperandStack.PopRef()
			}
			buf := extractBytes(srcRef)
			if buf != nil && offset >= 0 && length >= 0 && int(offset)+int(length) <= len(buf) {
				log.Printf("[JAVA-OUT] %s", string(buf[int(offset):int(offset)+int(length)]))
			}
		} else if strings.Contains(mDesc, "[B") {
			srcRef := frame.OperandStack.PopRef()
			if opcode != OpInvokestatic {
				_ = frame.OperandStack.PopRef()
			}
			buf := extractBytes(srcRef)
			if buf != nil {
				log.Printf("[JAVA-OUT] %s", string(buf))
			}
		} else {
			val := frame.OperandStack.PopInt()
			if opcode != OpInvokestatic {
				_ = frame.OperandStack.PopRef()
			}
			log.Printf("[JAVA-OUT] %c", rune(val))
		}
		return true
	case "flush", "close":
		if opcode != OpInvokestatic {
			_ = frame.OperandStack.PopRef()
		}
		return true
	case "checkError":
		if opcode != OpInvokestatic {
			_ = frame.OperandStack.PopRef()
		}
		frame.OperandStack.PushInt(0)
		return true
	case "<init>":
		paramCount := countMethodParamSlots(mDesc)
		for i := 0; i < paramCount; i++ {
			frame.OperandStack.PopSlot()
		}
		objRef := frame.OperandStack.PopRef()
		if obj, ok := objRef.(*Object); ok && obj != nil {
			obj.Extra = "PrintStream"
		}
		return true
	default:
		return false
	}
}

func javaIndexOfRune(s string, ch rune, from int32) int32 {
	runes := []rune(s)
	if from < 0 {
		from = 0
	}
	for i := int(from); i < len(runes); i++ {
		if runes[i] == ch {
			return int32(i)
		}
	}
	return -1
}

func javaIndexOfString(s, sub string, from int32) int32 {
	runes := []rune(s)
	subRunes := []rune(sub)
	if from < 0 {
		from = 0
	}
	if len(subRunes) == 0 {
		if int(from) > len(runes) {
			return int32(len(runes))
		}
		return from
	}
	for i := int(from); i+len(subRunes) <= len(runes); i++ {
		match := true
		for j := 0; j < len(subRunes); j++ {
			if runes[i+j] != subRunes[j] {
				match = false
				break
			}
		}
		if match {
			return int32(i)
		}
	}
	return -1
}

func extractString(ref interface{}) string {
	if ref == nil {
		return ""
	}
	if s, ok := ref.(string); ok {
		return s
	}
	if slot, ok := ref.(Slot); ok {
		if slot.Ref != nil {
			return extractString(slot.Ref)
		}
		if slot.Num != 0 {
			return fmt.Sprintf("%d", slot.Num)
		}
		return ""
	}
	if obj, ok := ref.(*Object); ok && obj != nil {
		if s, ok := obj.Extra.(string); ok {
			return s
		}
		if val := obj.GetField("value"); val.Ref != nil {
			if chars, ok := val.Ref.([]uint16); ok {
				return string(utf16.Decode(chars))
			}
			if bytes, ok := val.Ref.([]byte); ok {
				return string(bytes)
			}
		}
		for _, val := range obj.GetFieldsCopy() {
			if s, ok := val.Ref.(string); ok {
				return s
			}
		}
	}
	return fmt.Sprintf("%v", ref)
}

func extractBytes(ref interface{}) []byte {
	if ref == nil {
		return nil
	}
	if b, ok := ref.([]byte); ok {
		return b
	}
	if slots, ok := ref.([]Slot); ok {
		buf := make([]byte, len(slots))
		for i, s := range slots {
			buf[i] = byte(s.Num)
		}
		return buf
	}
	if obj, ok := ref.(*Object); ok && obj != nil {
		if b, ok := obj.Extra.([]byte); ok {
			return b
		}
		if slots, ok := obj.Extra.([]Slot); ok {
			buf := make([]byte, len(slots))
			for i, s := range slots {
				buf[i] = byte(s.Num)
			}
			return buf
		}
		for _, val := range obj.GetFieldsCopy() {
			if b, ok := val.Ref.([]byte); ok {
				return b
			}
			if slots, ok := val.Ref.([]Slot); ok {
				buf := make([]byte, len(slots))
				for i, s := range slots {
					buf[i] = byte(s.Num)
				}
				return buf
			}
		}
	}
	return nil
}

func extractImage(ref interface{}) image.Image {
	if ref == nil {
		return nil
	}
	if img, ok := ref.(image.Image); ok && img != nil {
		return img
	}
	if obj, ok := ref.(*Object); ok && obj != nil {
		if img, ok := obj.Extra.(image.Image); ok && img != nil {
			return img
		}
		obj.mu.RLock()
		defer obj.mu.RUnlock()
		for _, val := range obj.Fields {
			if val.Ref != nil && val.Ref != ref {
				if img, ok := val.Ref.(image.Image); ok && img != nil {
					return img
				}
				if subObj, ok := val.Ref.(*Object); ok && subObj != nil && subObj != ref {
					if img, ok := subObj.Extra.(image.Image); ok && img != nil {
						return img
					}
				}
			}
		}
	}
	return nil
}

func (ctx *VMContext) handleGraphicsCall(mName, mDesc string, objRef interface{}, args []Slot) bool {
	g := ctx.Graphics
	if gRef, ok := objRef.(*midp.Graphics); ok && gRef != nil {
		g = gRef
	} else if obj, ok := objRef.(*Object); ok && obj != nil {
		if gRef, ok := obj.Extra.(*midp.Graphics); ok && gRef != nil {
			g = gRef
		} else if gRef := extractGraphics(objRef, nil); gRef != nil {
			g = gRef
		}
	}

	if g == nil {
		return false
	}

	// Map obfuscated bx wrapper methods
	normName := mName
	if strings.Contains(mDesc, "Laq;") && countMethodParamSlots(mDesc) >= 8 {
		normName = "drawRegion"
	} else if strings.Contains(mDesc, "Laq;") {
		normName = "drawImage"
	} else if mName == "b" && mDesc == "(IIII)V" {
		normName = "fillRect"
	} else if mName == "c" && mDesc == "(IIII)V" {
		normName = "drawRect"
	} else if mName == "a" && mDesc == "(I)V" {
		normName = "setColor"
	} else if mName == "d" && mDesc == "(IIII)V" {
		normName = "setClip"
	} else if mName == "a" && mDesc == "(II)V" {
		normName = "translate"
	}

	allArgs := make([]Slot, 0, len(args)+1)
	allArgs = append(allArgs, Slot{Ref: objRef})
	allArgs = append(allArgs, args...)

	return handleGraphicsCall(g, normName, mDesc, allArgs)
}

func findImageInArgs(args []Slot) image.Image {
	if len(args) > 1 {
		for _, arg := range args[1:] {
			if img := extractImage(arg.Ref); img != nil {
				return img
			}
		}
	}
	for _, arg := range args {
		if img := extractImage(arg.Ref); img != nil {
			return img
		}
	}
	return nil
}

func getIntArgs(args []Slot) []int {
	var res []int
	for _, a := range args {
		if a.Ref == nil {
			res = append(res, int(a.Num))
		}
	}
	return res
}

func handleGraphicsCall(g *midp.Graphics, mName, mDesc string, args []Slot) bool {
	// 1. setColor: (III)V or (I)V
	if mName == "setColor" {
		nums := getIntArgs(args)
		if len(nums) >= 3 {
			g.SetColor(nums[0], nums[1], nums[2])
			return true
		} else if len(nums) >= 1 {
			rgb := nums[0]
			r := (rgb >> 16) & 0xFF
			gVal := (rgb >> 8) & 0xFF
			b := rgb & 0xFF
			g.SetColor(r, gVal, b)
			return true
		}
	}

	// 2. translate(int, int)
	if mName == "translate" {
		nums := getIntArgs(args)
		if len(nums) >= 2 {
			g.Translate(nums[0], nums[1])
			return true
		}
	}

	// 3. fillRect(x, y, w, h)
	if mName == "fillRect" {
		nums := getIntArgs(args)
		if len(nums) >= 4 {
			g.FillRect(nums[0], nums[1], nums[2], nums[3])
			return true
		}
	}

	// 4. setClip / clipRect
	if mName == "setClip" {
		nums := getIntArgs(args)
		if len(nums) >= 4 {
			w := nums[2]
			h := nums[3]
			if w <= 0 {
				w = g.Width
			}
			if h <= 0 {
				h = g.Height
			}
			g.SetClip(nums[0], nums[1], w, h)
			return true
		}
	}
	if mName == "clipRect" {
		nums := getIntArgs(args)
		if len(nums) >= 4 {
			g.ClipRect(nums[0], nums[1], nums[2], nums[3])
			return true
		}
	}

	// 5. drawRect / drawLine
	if mName == "drawRect" || mName == "drawLine" {
		nums := getIntArgs(args)
		if len(nums) >= 4 {
			g.DrawRect(nums[0], nums[1], nums[2], nums[3])
			return true
		}
	}

	// 6. drawString
	if mName == "drawString" || strings.Contains(mDesc, "String") {
		var str string
		for _, a := range args {
			s := extractString(a.Ref)
			if s != "" {
				str = s
				break
			}
		}
		nums := getIntArgs(args)
		if len(nums) >= 2 {
			anchor := 0
			if len(nums) >= 3 {
				anchor = nums[2]
			}
			g.DrawString(str, nums[0], nums[1], anchor)
			return true
		}
	}

	// 7. drawImage / bx.a
	if realImg := findImageInArgs(args); realImg != nil {
		nums := getIntArgs(args)
		if len(nums) >= 8 {
			// DrawRegion: srcX, srcY, w, h, transform, destX, destY, anchor
			anchor := 0
			if len(nums) >= 8 {
				anchor = nums[7]
			}
			g.DrawRegion(realImg, nums[0], nums[1], nums[2], nums[3], nums[4], nums[5], nums[6], anchor)
			return true
		} else if len(nums) >= 2 {
			// DrawImage: destX, destY, anchor
			anchor := 0
			if len(nums) >= 3 {
				anchor = nums[2]
			}
			g.DrawImage(realImg, nums[0], nums[1], anchor)
			return true
		}
	}

	return false
}

func (ctx *VMContext) NewString(str string) *Object {
	var cf *classfile.ClassFile
	if ctx.ClassLoader != nil {
		cf, _ = ctx.ClassLoader.LoadClass("java/lang/String")
	}
	obj := NewObject(cf)
	obj.Extra = str
	return obj
}

func (ctx *VMContext) NewCommand(label string, cmdType, priority int32) *Object {
	cf, _ := ctx.ClassLoader.LoadClass("javax/microedition/lcdui/Command")
	cmd := NewObject(cf)
	cmd.SetField("label", Slot{Ref: ctx.NewString(label)})
	cmd.SetField("cmdType", Slot{Num: cmdType})
	cmd.SetField("priority", Slot{Num: priority})
	return cmd
}

func (ctx *VMContext) ExtractString(ref interface{}) string {
	return extractString(ref)
}

func transformImageRegion(img image.Image, xSrc, ySrc, w, h, transform int) image.Image {
	if img == nil || w <= 0 || h <= 0 {
		return nil
	}
	outW, outH := w, h
	switch transform {
	case 4, 5, 6, 7:
		outW, outH = h, w
	}
	transImg := image.NewRGBA(image.Rect(0, 0, outW, outH))
	srcBounds := img.Bounds()
	for sy := 0; sy < h; sy++ {
		for sx := 0; sx < w; sx++ {
			srcX := xSrc + sx
			srcY := ySrc + sy
			if srcX < srcBounds.Min.X || srcX >= srcBounds.Max.X || srcY < srcBounds.Min.Y || srcY >= srcBounds.Max.Y {
				continue
			}
			dx, dy := sx, sy
			switch transform {
			case 0: // TRANS_NONE
				dx, dy = sx, sy
			case 1: // TRANS_MIRROR_ROT180 (Vertical flip)
				dx, dy = sx, h-1-sy
			case 2: // TRANS_MIRROR (Horizontal flip)
				dx, dy = w-1-sx, sy
			case 3: // TRANS_ROT180 (180 degree rotation)
				dx, dy = w-1-sx, h-1-sy
			case 4: // TRANS_MIRROR_ROT270
				dx, dy = sy, sx
			case 5: // TRANS_ROT90
				dx, dy = h-1-sy, sx
			case 6: // TRANS_ROT270
				dx, dy = sy, w-1-sx
			case 7: // TRANS_MIRROR_ROT90
				dx, dy = h-1-sy, w-1-sx
			}
			if dx >= 0 && dx < outW && dy >= 0 && dy < outH {
				transImg.Set(dx, dy, img.At(srcX, srcY))
			}
		}
	}
	return transImg
}

func rgbAlpha(pixel int32, processAlpha bool) uint8 {
	if !processAlpha {
		// J2ME treats RGB pixels as opaque when processAlpha is false. Keeping
		// the high byte in that mode turns ordinary RGB data into transparent
		// layers, which makes character/equipment renders disappear.
		return 0xFF
	}
	return uint8((pixel >> 24) & 0xFF)
}
