package vm

import (
	"log"
	"sort"
	"sync"

	"java-app/pkg/classfile"
	"java-app/pkg/jar"
)

// IdentityXORMethods is the list of ClassName.method:desc keys identified as
// Knight Online stream-cipher XOR helpers. Filled by ScanXORMethods plus any
// RegisterIdentityXOR fallbacks. SocketTap owns the real cipher; these natives
// must return the input byte unchanged.
var IdentityXORMethods []string

var (
	identityXORMu  sync.Mutex
	identityXORSet = make(map[string]struct{})
)

func xorIdentityKey(clsName, mName, mDesc string) string {
	return clsName + "." + mName + ":" + mDesc
}

func init() {
	// KnightOnline_402.jar: send XOR is instance l.a:(B)B (field s);
	// recv XOR is static l.a:(Ll;B)B (field r). Scan only matches (B)B/(I)I/(B)I,
	// so the static accessor is a hardcoded fallback.
	// q.b(byte,short,byte) is the packet writer, not the cipher — ActionCommand stays 34.
	RegisterIdentityXOR("l.a:(B)B")
	RegisterIdentityXOR("l.a:(Ll;B)B")
}

// RegisterIdentityXOR records a ClassName.method:desc key as an identity XOR stub.
func RegisterIdentityXOR(key string) {
	identityXORMu.Lock()
	defer identityXORMu.Unlock()
	registerIdentityXORLocked(key)
}

func registerIdentityXORLocked(key string) {
	if _, ok := identityXORSet[key]; ok {
		return
	}
	identityXORSet[key] = struct{}{}
	IdentityXORMethods = append(IdentityXORMethods, key)
}

// IsIdentityXOR reports whether the given native should return its last integer
// argument unchanged instead of running the JAR XOR.
func IsIdentityXOR(clsName, mName, mDesc string) bool {
	key := xorIdentityKey(clsName, mName, mDesc)
	identityXORMu.Lock()
	defer identityXORMu.Unlock()
	_, ok := identityXORSet[key]
	return ok
}

// ScanXORMethods finds (B)B / (I)I / (B)I methods whose bytecode looks like a
// table-driven XOR stream cipher (ixor + array load + field access).
func ScanXORMethods(jarBytes []byte) []string {
	archive, err := jar.ReadJar(jarBytes)
	if err != nil {
		log.Printf("[xorstub] ReadJar: %v", err)
		return nil
	}

	var found []string
	for className, data := range archive.Classes {
		cf, err := classfile.Parse(data)
		if err != nil {
			continue
		}
		cls := cf.GetClassName()
		if cls == "" {
			cls = className
		}
		for i := range cf.Methods {
			m := &cf.Methods[i]
			if len(m.Code) == 0 {
				continue
			}
			mName := cf.ConstantPool.GetUtf8(m.NameIndex)
			mDesc := cf.ConstantPool.GetUtf8(m.DescIndex)
			if mDesc != "(B)B" && mDesc != "(I)I" && mDesc != "(B)I" {
				continue
			}
			if !codeLooksLikeXORCipher(m.Code) {
				continue
			}
			found = append(found, xorIdentityKey(cls, mName, mDesc))
		}
	}
	sort.Strings(found)

	identityXORMu.Lock()
	for _, key := range found {
		registerIdentityXORLocked(key)
	}
	identityXORMu.Unlock()

	log.Printf("[xorstub] methods=%v", found)
	return found
}

func codeLooksLikeXORCipher(code []byte) bool {
	hasIxor := false
	hasAload := false
	hasGet := false
	for _, b := range code {
		switch b {
		case OpIxor:
			hasIxor = true
		case OpBaload, OpIaload:
			hasAload = true
		case OpGetfield, OpGetstatic:
			hasGet = true
		}
	}
	return hasIxor && hasAload && hasGet
}
