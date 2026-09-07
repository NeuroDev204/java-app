# Knight Online J2ME Hashtable & RMS Cache Design Specification

**Date:** 2026-09-05  
**Topic:** Fixing Missing Render Data & Stuck "Đang lấy dữ liệu" Dialog in Knight Online J2ME Emulator  
**Author:** AI Pair Programming Assistant & Senior Software Engineer  

---

## 1. Executive Summary & Root Cause Analysis

### 1.1 Problem Statement
When running `KnightOnline_402.jar` on the Android emulator:
1. Map environment, player character, and text render, but certain sprites (trees, NPC icons, equipment icons) appear missing or replaced with placeholder fallback boxes.
2. Interacting with NPCs (e.g., Mrs Oda, Aman) displays the modal dialog **"Đang lấy dữ liệu"** (Loading data) with an **"Ok"** button, which remains permanently stuck.
3. Log analysis (`/tmp/knight-render-trace.log`) reveals over 132,000 `[IMAGE-NOT-FOUND]`, 4,780 `[IMAGE-BYTES-FALLBACK]`, and over 9,460 `[SOCK-WRITE]` calls in under 3 seconds, culminating in:
   `[SOCK-READ-ERR] closed connection on err=read tcp ...: connection reset by peer`.

### 1.2 Root Cause Chain
1. **Unimplemented `java.util.Hashtable` & `java.util.Enumeration` in Go VM:**
   - Knight Online encapsulates its entire memory sprite cache in `el.class` using `java.util.Hashtable` (`cn.o` for NPCs, `fq.b` for trees/scenery, `cr.a` for effects, `dr.b` for items, `bw.aj` for mounts).
   - In Go VM (`pkg/vm/interpreter.go`), `java/util/Hashtable` falls through to the generic native stub: `get()` returns `null`, `put()` is a no-op, and `containsKey()` returns `false`.
   - **Result:** No sprite is ever cached in memory. Every frame, the game queries `el.a(key)`, receives `null`, and instantiates a new sprite descriptor `fe`.

2. **Flawed `RecordStore` (RMS) Miss Handling:**
   - When a sprite is not found in memory, `bk.a(short)` queries RMS via `ak.a("image" + id)`, which calls `openRecordStore(name, false)`.
   - In standard MIDP RMS, `openRecordStore` with `createIfNecessary=false` must throw `RecordStoreNotFoundException` if the store does not exist, and `getRecord(1)` on an empty store must throw `InvalidRecordIDException`.
   - The Go VM implementation creates an empty store with `records=0` and returns a typed nil slice `[]byte(nil)`.
   - In Go, `Slot{Ref: []byte(nil)}` is a non-nil interface (`ref != nil` is true for bytecode `ifnull`), so Java receives a 0-byte slice rather than `null`.
   - `Image.createImage([BII)` decodes 0 bytes, creating a fallback dummy image on every frame.

3. **Bypassed Throttle & Packet Flooding:**
   - Java class `bk.a(short)` contains a 200-frame throttle (`fe.e >= 200`) before re-requesting an image from the server.
   - Because `fe` is never cached in `Hashtable`, every frame constructs a fresh `fe` with `fe.e = 0`, bypassing the throttle completely.
   - The client floods packet `-51` / `-52` to the game server on every frame for every visible entity (600+ packets/sec).
   - The game server detects the flood and resets the TCP connection (`RST`).
   - Subsequent user actions (talking to an NPC via packet 15) fail silently because the socket is dead, leaving "Đang lấy dữ liệu" open forever.

4. **Unloaded Pre-existing RMS Cache:**
   - The project repository contains `rms_data.json` with 90 pre-downloaded images (`image3005`, `image3023`, `image3029`, `image3044`, etc.).
   - The Go VM initializes `ctx.RecordStores` as an empty map and never loads `rms_data.json`.

---

## 2. Architecture & Design Solutions

```
+-------------------------------------------------------------------------+
|                              Knight Online J2ME                         |
|  el.class (Hashtable wrapper)  <-->  bk.class (Sprite cache manager)    |
+------------------------------------+------------------------------------+
                                     |
                    +----------------+----------------+
                    |                                 |
                    v                                 v
+---------------------------------------+  +------------------------------+
|       pkg/vm/hashtable.go             |  |      pkg/vm/rms.go           |
| - HashtableData (sync.RWMutex)        |  | - Preloads default_rms.json  |
|   * get, put, containsKey, remove     |  | - Throws RecordStoreNotFound |
|   * keys(), elements() Enumeration    |  | - Throws InvalidRecordID     |
| - Normalized key comparison (String)  |  | - Normalizes nil []byte      |
+---------------------------------------+  +------------------------------+
                    |                                 |
                    +----------------+----------------+
                                     |
                                     v
+-------------------------------------------------------------------------+
|                  pkg/vm/interpreter.go (Execution Engine)               |
| - OperandStack.PushRef normalizes nil slices to Slot{Ref: nil}          |
| - Resolves java/util/Hashtable & java/util/Enumeration natives          |
| - Cache hit in Hashtable: 0 network requests after first frame          |
+-------------------------------------------------------------------------+
```

### 2.1 Native `java/util/Hashtable` & `java/util/Enumeration` (`pkg/vm/hashtable.go`)

We implement thread-safe backing structures for `java.util.Hashtable`:

```go
type HashtableData struct {
    mu      sync.RWMutex
    entries map[interface{}]interface{}
    keys    []interface{} // Preserves insertion/enumeration order
}
```

#### Key Normalization
In Java J2ME bytecode, keys are typically `java.lang.String` (wrapped in Go as `string` or `*Object{Extra: string}`) or `java.lang.Integer`.
We define `normalizeHashKey(ref interface{}) interface{}`:
- If `s, ok := ref.(string)` -> return `s`.
- If `obj, ok := ref.(*Object)`:
  - If `obj.Extra` is a `string` -> return `obj.Extra`.
  - If `obj` has a field `value` (e.g. `java.lang.Integer`) -> return `val`.
- Otherwise return `ref`.

#### Supported Methods:
1. `<init>()V`, `<init>(I)V`, `<init>(IF)V` -> Allocate `NewHashtableData()`.
2. `put(Ljava/lang/Object;Ljava/lang/Object;)Ljava/lang/Object;` -> Inserts key-value pair, returns previous value or `nil`.
3. `get(Ljava/lang/Object;)Ljava/lang/Object;` -> Returns value or `nil`.
4. `containsKey(Ljava/lang/Object;)Z` -> Returns 1 if present, 0 otherwise.
5. `contains(Ljava/lang/Object;)Z` -> Returns 1 if value present, 0 otherwise.
6. `remove(Ljava/lang/Object;)Ljava/lang/Object;` -> Removes key, returns old value.
7. `clear()V` -> Clears all entries.
8. `size()I` -> Returns entry count.
9. `isEmpty()Z` -> Returns 1 if size == 0, 0 otherwise.
10. `keys()Ljava/util/Enumeration;` -> Returns `Object{Extra: &EnumerationData{items: keys}}`.
11. `elements()Ljava/util/Enumeration;` -> Returns `Object{Extra: &EnumerationData{items: values}}`.

#### Native `java/util/Enumeration`:
```go
type EnumerationData struct {
    mu     sync.Mutex
    items  []interface{}
    cursor int
}
```
- `hasMoreElements()Z`: Returns 1 if `cursor < len(items)`, else 0.
- `nextElement()Ljava/lang/Object;`: Returns `items[cursor++]` or `nil` if out of bounds.

---

### 2.2 RecordStore (RMS) Standard Compliance & Exception Handling

In `pkg/vm/interpreter.go`:

1. **`openRecordStore(String name, boolean createIfNecessary)`:**
   ```go
   store, exists := ctx.RecordStores[name]
   if !exists {
       if createIfNecessary == 0 {
           frame.PendingException = ctx.NewException("javax/microedition/rms/RecordStoreNotFoundException", "RecordStore not found: "+name)
           return
       }
       store = &RecordStoreData{Records: make([][]byte, 0)}
       ctx.RecordStores[name] = store
   }
   ```
2. **`getRecord(int recordId)`:**
   ```go
   idx := int(recID) - 1
   if idx < 0 || idx >= len(store.Records) {
       frame.PendingException = ctx.NewException("javax/microedition/rms/InvalidRecordIDException", "Invalid record id")
       return
   }
   result := store.Records[idx]
   if len(result) == 0 {
       frame.OperandStack.PushRef(nil)
   } else {
       frame.OperandStack.PushRef(result)
   }
   ```
3. **`OperandStack.PushRef` nil normalization in `pkg/vm/frame.go`:**
   ```go
   func (s *OperandStack) PushRef(ref interface{}) {
       if s.size >= uint(len(s.slots)) {
           s.slots = append(s.slots, Slot{})
       }
       if b, ok := ref.([]byte); ok && b == nil {
           ref = nil
       }
       s.slots[s.size] = Slot{Ref: ref}
       s.size++
   }
   ```

---

### 2.3 Embedded Default RMS Asset Preloading (`pkg/vm/rms.go`)

1. Copy `rms_data.json` to `pkg/vm/default_rms.json`.
2. Use Go 1.16+ embed:
   ```go
   //go:embed default_rms.json
   var defaultRMSData []byte
   ```
3. In `NewVMContext`, call `ctx.loadDefaultRMS()`:
   - Parses the 90 record stores and populates `ctx.RecordStores`.
   - `image3005`, `image3023`, `image3029`, `image3044`, etc. are immediately available offline.
   - On Android and iOS, no external file read is required; the game boots with full textures available instantly.

---

## 3. Data Flow & Impact Analysis

| Event | Previous Broken Flow | New Fixed Flow |
|---|---|---|
| **Render Map Entities** | `el.a(key)` returns `null` every frame -> `fe` recreated -> `openRecordStore` returns 0-byte slice -> `Image.createImage` fails -> Spams `-51`/`-52` -> Server disconnects | `el.a(key)` returns cached `fe` -> Image already in `fe.a` -> Render draws immediately -> 0 network spam |
| **RMS Lookup for Images** | `openRecordStore` creates empty store, returns 0-byte slice | Preloaded `default_rms.json` hits valid image data -> Decodes valid bitmap -> Cached in `fe.a` |
| **Talking to NPC** | Client sends packet 15 over dead socket -> Dialog "Đang lấy dữ liệu" hangs forever | Socket remains alive -> Server responds with dialogue/menu -> Dialog dismissed cleanly |

---

## 4. Verification & Testing Plan

1. **Automated Go Unit Tests:**
   - `pkg/vm/hashtable_test.go`:
     - Test basic `put`, `get`, `remove`, `containsKey`, `clear`.
     - Test key normalization (Go strings vs Java String Object).
     - Test `keys()` and `elements()` with `java.util.Enumeration`.
     - Test concurrent read/writes with goroutines.
   - `pkg/vm/rms_test.go`:
     - Test `RecordStoreNotFoundException` when `createIfNecessary = false`.
     - Test `InvalidRecordIDException` when record id out of range.
     - Test preloaded `default_rms.json` populates `image3029` with valid bytes.
   - `pkg/vm/frame_test.go`:
     - Test `PushRef(nil []byte)` results in `ref == nil`.
2. **Integration Verification on Android Emulator:**
   - Recompile gomobile AAR via `build_aar.sh`.
   - Deploy debug APK to `emulator-5554`.
   - Log into game -> Enter world.
   - Confirm in logcat:
     - `[RMS] Opened RecordStore` hits preloaded records (`records=1`).
     - `[IMAGE-BYTES-FALLBACK]` count drops to 0.
     - `[SOCK-WRITE]` spam ceases; TCP connection remains open.
   - Tap NPC "Mrs Oda" -> Verify dialogue menu opens without hanging on "Đang lấy dữ liệu".
