// Package gamebridge exposes a gomobile-compatible API for loading and running
// a J2ME MIDlet JAR on iOS/Android via the pure-Go VM interpreter.
//
// The surface is intentionally minimal so that gomobile bind can generate
// clean Objective-C / Java bindings without CGO:
//
//	Bridge.LoadJAR(jarBytes []byte) string   // "" on success, error message otherwise
//	Bridge.Tick()                            // advance one frame
//	Bridge.GetFrameBuffer() []byte           // RGBA pixel bytes (width×height×4)
//	Bridge.GetWidth() int
//	Bridge.GetHeight() int
//	Bridge.SendTouchDown(x, y int)
//	Bridge.SendTouchDrag(x, y int)
//	Bridge.SendTouchUp(x, y int)
//	Bridge.SendKeyPress(keyCode int)
//	Bridge.SendKeyDown(keyCode int)
//	Bridge.SendKeyUp(keyCode int)
//	Bridge.SetShouldPaint(paint bool)
//	Bridge.ShouldPaint() bool
//	Bridge.StartUpgrade(configJSON string) string
//	Bridge.StopUpgrade()
//	Bridge.GetUpgradeStatus() string
//	Bridge.Stop()
package gamebridge

import (
	"encoding/json"
	"fmt"
	"image/draw"
	"java-app/pkg/classfile"
	"java-app/pkg/jar"
	"java-app/pkg/midp"
	"java-app/pkg/network/protocol"
	"java-app/pkg/upgradebot"
	"java-app/pkg/vm"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// canvasWidth / canvasHeight match the classic J2ME 240×320 QVGA viewport
	// used by KnightOnline_402.jar.
	canvasWidth  = 240
	canvasHeight = 320

	emptyUpgradeStatusJSON = `{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}`
)

// touchEvent represents a single pointer press, drag, or release at (x, y).
type touchEvent struct {
	x, y    int
	pressed bool
	dragged bool
}

// tapInjector implements upgradebot.Injector via the live SocketTap.
type tapInjector struct {
	b *Bridge
}

func (t tapInjector) Inject(msg *protocol.Message) error {
	if t.b == nil {
		return fmt.Errorf("not connected")
	}
	t.b.mu.Lock()
	var tap *midp.SocketTap
	if t.b.vmCtx != nil {
		tap = t.b.vmCtx.ActiveSocket
	}
	t.b.mu.Unlock()
	if tap == nil {
		return fmt.Errorf("not connected")
	}
	return tap.Inject(msg)
}

// Bridge is the singleton game engine instance exposed to gomobile consumers.
// Only one instance may exist per process because the VM interpreter uses
// package-level globals for its classfile cache.
type Bridge struct {
	mu sync.Mutex

	gfx     *midp.Graphics
	vmCtx   *vm.VMContext
	archive *jar.JarArchive

	// mainClass is the MIDlet entry-point resolved from META-INF/MANIFEST.MF.
	mainClass string

	// midletCF is the loaded ClassFile for the MIDlet, cached for reuse.
	midletCF *classfile.ClassFile

	// midletObj is the singleton MIDlet instance object passed to every call.
	midletObj *vm.Object

	// started tracks whether startApp() was already invoked on the MIDlet.
	started bool

	// shouldPaint is a Flutter lifecycle hint. The VM loop always runs;
	// GetFrameBuffer still returns the last copy when this is false.
	shouldPaint bool

	bot      *upgradebot.Controller
	wiredTap *midp.SocketTap

	// touchQueue is a channel of pending touch events delivered to the VM loop.
	touchQueue chan touchEvent

	// stopCh is closed to signal the background tick goroutine to halt.
	stopCh chan struct{}
	wg     sync.WaitGroup

	// frameBuffer is a pre-allocated RGBA byte slice returned to callers.
	// Re-using the allocation avoids repeated GC pressure from the Dart FFI layer.
	frameBuffer      []byte
	lastRevision     uint64
	lastSentRevision uint64
}

// SetDataDir gives the engine a writable directory in which to keep the MIDP
// record store across launches. Call it before LoadJAR. Without it record
// stores live in memory only, so everything the game downloads while playing —
// item icons in particular — is discarded on exit and must be fetched again,
// leaving inventory slots on the game's loading spinner. Pass "" to disable.
func SetDataDir(dir string) {
	if dir == "" {
		vm.SetRMSPath("")
		return
	}
	vm.SetRMSPath(filepath.Join(dir, "rms.json"))
}

// NewBridge constructs an uninitialised Bridge. Call LoadJAR to boot the VM.
func NewBridge() *Bridge {
	return &Bridge{
		touchQueue:  make(chan touchEvent, 64),
		stopCh:      make(chan struct{}),
		frameBuffer: make([]byte, canvasWidth*canvasHeight*4),
		shouldPaint: true,
	}
}

// LoadJAR parses jarBytes, initialises the VM, and calls startApp() on the
// MIDlet. It returns an empty string on success or a human-readable error
// description on failure.
//
// This method is synchronous and may take a few hundred milliseconds on first
// call because the interpreter initialises all static class fields.
func (b *Bridge) LoadJAR(jarBytes []byte) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return "already started"
	}

	// --- Parse the JAR archive ---
	archive, err := jar.ReadJar(jarBytes)
	if err != nil {
		return fmt.Sprintf("ReadJar: %v", err)
	}
	b.archive = archive

	mainClass := archive.MainMIDletClass
	if mainClass == "" {
		// Fallback: well-known class name for TeamObi Knight Online
		mainClass = "com/silverknight/TemMidlet"
	}
	b.mainClass = mainClass
	log.Printf("[bridge] main MIDlet class: %s", mainClass)

	// Register XOR identity stubs before startApp — login I/O can happen there.
	for _, key := range vm.ScanXORMethods(jarBytes) {
		vm.RegisterIdentityXOR(key)
	}

	// --- Initialise graphics canvas ---
	b.gfx = midp.NewGraphics(canvasWidth, canvasHeight)

	// --- Initialise VM context ---
	cl := classfile.NewClassLoader(archive)
	b.vmCtx = vm.NewVMContext(cl, b.gfx)

	// --- Load and boot the MIDlet ---
	cf, err := cl.LoadClass(mainClass)
	if err != nil {
		return fmt.Sprintf("LoadClass(%s): %v", mainClass, err)
	}
	b.midletCF = cf

	// Create the singleton MIDlet object used for all subsequent method calls.
	b.midletObj = vm.NewObject(cf)

	var bootErr error
	b.vmCtx.CallSerialized(func() {
		// Run static initializer (<clinit>) to set up class-level state.
		b.vmCtx.EnsureClassInitialized(cf)

		// Call the no-arg constructor to initialise instance fields.
		ctor := cf.GetMethod("<init>", "()V")
		if ctor != nil {
			if _, err = b.vmCtx.InvokeMethod(cf, ctor, []vm.Slot{{Ref: b.midletObj}}); err != nil {
				log.Printf("[bridge] <init> error (non-fatal): %v", err)
			}
		}

		// Call startApp() — the MIDlet lifecycle entry point.
		startApp := cf.GetMethod("startApp", "()V")
		if startApp == nil {
			bootErr = fmt.Errorf("startApp not found in %s", mainClass)
			return
		}
		if _, err = b.vmCtx.InvokeMethod(cf, startApp, []vm.Slot{{Ref: b.midletObj}}); err != nil {
			log.Printf("[bridge] startApp error (non-fatal): %v", err)
		}
	})
	if bootErr != nil {
		return bootErr.Error()
	}

	b.maybeWireTapLocked()
	b.started = true

	// Flush the initial frame painted during startApp into the front-buffer.
	b.gfx.Flush()
	b.copyFrameBuffer()

	// Start background tick goroutine.
	b.wg.Add(1)
	go b.runLoop()

	return ""
}

// runLoop drives the game's paint/run cycle in a background goroutine.
// It ticks at ~20 fps which is the typical J2ME game cadence.
func (b *Bridge) runLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond) // 20 fps
	defer ticker.Stop()

	for {
		b.maybeWireTap()
		select {
		case <-b.stopCh:
			return
		case ev := <-b.touchQueue:
			b.dispatchTouch(ev)
		case <-ticker.C:
			b.paintFrame()
		}
	}
}

// maybeWireTap attaches bot.OnMessage to ActiveSocket once the tap appears.
func (b *Bridge) maybeWireTap() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maybeWireTapLocked()
}

func (b *Bridge) maybeWireTapLocked() {
	if b.vmCtx == nil {
		return
	}
	tap := b.vmCtx.ActiveSocket
	if tap == b.wiredTap {
		return
	}
	// Drop or replace: Task 3 Start does not restore connected after NotifyDisconnected,
	// and the old snapshot's in_game must not leak onto a new TCP session.
	if b.wiredTap != nil {
		if b.bot != nil {
			b.bot.NotifyDisconnected()
		}
		b.wiredTap = nil
		if tap == nil {
			return
		}
	}
	if tap == nil {
		return
	}
	b.bot = upgradebot.NewController(tapInjector{b})
	tap.SetOnPlainMessage(b.bot.OnMessage)
	b.wiredTap = tap
}

// paintFrame flushes any pending repaints and updates the front-buffer.
func (b *Bridge) paintFrame() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.vmCtx == nil || b.midletCF == nil || b.midletObj == nil {
		return
	}

	b.maybeWireTapLocked()
	if b.vmCtx.RepaintPending {
		var (
			canvasObj *vm.Object
			canvasCF  *classfile.ClassFile
		)
		if canvas := b.vmCtx.CurrentCanvas; canvas != nil && canvas.Class != nil {
			canvasObj = canvas
			canvasCF = canvas.Class
		} else if b.midletObj != nil && b.midletCF != nil {
			canvasObj = b.midletObj
			canvasCF = b.midletCF
		}
		if canvasObj != nil && canvasCF != nil && b.gfx != nil {
			b.vmCtx.RepaintPending = false
			b.gfx.BeginPaint()
			paintM := canvasCF.GetMethod("paint", "(Ljavax/microedition/lcdui/Graphics;)V")
			if paintM != nil {
				b.vmCtx.CallSerialized(func() {
					_, _ = b.vmCtx.InvokeMethod(canvasCF, paintM, []vm.Slot{
						{Ref: canvasObj},
						{Ref: b.gfx},
					})
				})
			}
			b.gfx.EndPaint()
		}
	} else if b.gfx != nil {
		b.gfx.Flush()
	}
	b.copyFrameBuffer()
}

// dispatchTouch delivers a pointer event to the current LCDUI Canvas.
// Fall back to the MIDlet only when Display.setCurrent has not installed a canvas.
func (b *Bridge) dispatchTouch(ev touchEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.vmCtx == nil {
		return
	}

	methodName := "pointerReleased"
	if ev.dragged {
		methodName = "pointerDragged"
	} else if ev.pressed {
		methodName = "pointerPressed"
	}

	var (
		target *vm.Object
		cf     *classfile.ClassFile
	)
	if canvas := b.vmCtx.CurrentCanvas; canvas != nil && canvas.Class != nil {
		target = canvas
		cf = canvas.Class
	} else if b.midletObj != nil && b.midletCF != nil {
		target = b.midletObj
		cf = b.midletCF
	} else {
		return
	}

	targetCf, method := b.vmCtx.ResolveVirtualMethod(target, methodName, "(II)V")
	if method == nil {
		method = cf.GetMethod(methodName, "(II)V")
		targetCf = cf
	}
	if method == nil {
		// Fallback 1: check if midletObj has the method
		if b.midletObj != nil && b.midletCF != nil && target != b.midletObj {
			target = b.midletObj
			cf = b.midletCF
			targetCf, method = b.vmCtx.ResolveVirtualMethod(target, methodName, "(II)V")
			if method == nil {
				method = cf.GetMethod(methodName, "(II)V")
				targetCf = cf
			}
		}
		// Fallback 2: check TemMidlet.a (the game Canvas instance)
		if method == nil && b.vmCtx != nil {
			slot := b.vmCtx.GetStaticField("com/silverknight/TemMidlet.a")
			if aTarget, ok := slot.Ref.(*vm.Object); ok && aTarget != nil && aTarget.Class != nil {
				target = aTarget
				targetCf, method = b.vmCtx.ResolveVirtualMethod(target, methodName, "(II)V")
				if method == nil {
					method = aTarget.Class.GetMethod(methodName, "(II)V")
					targetCf = aTarget.Class
				}
			}
		}
	}
	if method == nil {
		return
	}

	log.Printf("[TOUCH] dispatch: %s(%d, %d) on %s", methodName, ev.x, ev.y, targetCf.GetClassName())

	args := []vm.Slot{
		{Ref: target},
		{Num: int32(ev.x)},
		{Num: int32(ev.y)},
	}
	b.vmCtx.CallSerialized(func() {
		_, _ = b.vmCtx.InvokeMethod(targetCf, method, args)
	})
	b.maybeWireTapLocked()
}

// dispatchKey delivers a key event (keyPressed or keyReleased) to the current LCDUI Canvas.
func (b *Bridge) dispatchKey(keyCode int, pressed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastSentRevision = 0
	if b.vmCtx == nil {
		return
	}

	methodName := "keyReleased"
	if pressed {
		methodName = "keyPressed"
	}

	var (
		target *vm.Object
		cf     *classfile.ClassFile
	)
	if canvas := b.vmCtx.CurrentCanvas; canvas != nil && canvas.Class != nil {
		target = canvas
		cf = canvas.Class
	} else if b.midletObj != nil && b.midletCF != nil {
		target = b.midletObj
		cf = b.midletCF
	} else {
		return
	}

	targetCf, method := b.vmCtx.ResolveVirtualMethod(target, methodName, "(I)V")
	if method == nil {
		method = cf.GetMethod(methodName, "(I)V")
		targetCf = cf
	}
	if method == nil {
		// Fallback 1: check if midletObj has the method
		if b.midletObj != nil && b.midletCF != nil && target != b.midletObj {
			target = b.midletObj
			cf = b.midletCF
			targetCf, method = b.vmCtx.ResolveVirtualMethod(target, methodName, "(I)V")
			if method == nil {
				method = cf.GetMethod(methodName, "(I)V")
				targetCf = cf
			}
		}
		// Fallback 2: check TemMidlet.a (the game Canvas instance)
		if method == nil && b.vmCtx != nil {
			slot := b.vmCtx.GetStaticField("com/silverknight/TemMidlet.a")
			if aTarget, ok := slot.Ref.(*vm.Object); ok && aTarget != nil && aTarget.Class != nil {
				target = aTarget
				targetCf, method = b.vmCtx.ResolveVirtualMethod(target, methodName, "(I)V")
				if method == nil {
					method = aTarget.Class.GetMethod(methodName, "(I)V")
					targetCf = aTarget.Class
				}
			}
		}
	}
	if method == nil {
		return
	}

	log.Printf("[KEY] dispatch: %s(%d) on %s", methodName, keyCode, targetCf.GetClassName())

	args := []vm.Slot{
		{Ref: target},
		{Num: int32(keyCode)},
	}
	b.vmCtx.CallSerialized(func() {
		_, _ = b.vmCtx.InvokeMethod(targetCf, method, args)
	})
	b.maybeWireTapLocked()
}

// copyFrameBuffer copies the Graphics front-buffer RGBA pixels into b.frameBuffer.
// Must be called with b.mu held.
func (b *Bridge) copyFrameBuffer() {
	if b.gfx == nil || b.gfx.FrontBuffer == nil {
		return
	}
	b.gfx.RLock()
	rev := b.gfx.Revision
	if rev == b.lastRevision && len(b.frameBuffer) > 0 {
		b.gfx.RUnlock()
		return
	}
	b.lastRevision = rev
	src := b.gfx.FrontBuffer.Pix
	if len(src) == len(b.frameBuffer) {
		copy(b.frameBuffer, src)
	} else {
		b.frameBuffer = make([]byte, len(src))
		copy(b.frameBuffer, src)
	}
	b.gfx.RUnlock()
}

// --- Public gomobile-compatible API ---

// Tick manually advances the game loop by one frame.
func (b *Bridge) Tick() {
	b.paintFrame()
}

// GetFrameBuffer returns the most recent RGBA pixel buffer as a byte slice.
// Length = Width × Height × 4 (interleaved R,G,B,A).
// A fresh copy is returned each call so the Dart isolate can read it safely.
func (b *Bridge) GetFrameBuffer() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]byte, len(b.frameBuffer))
	copy(out, b.frameBuffer)
	return out
}

// GetFrameBufferIfChanged returns the RGBA pixel buffer only when new frames
// have been rendered since the last call, or nil if no visual change occurred.
// This eliminates ~6 MB/s of redundant memory allocations on mobile.
func (b *Bridge) GetFrameBufferIfChanged() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.lastRevision == b.lastSentRevision && b.lastSentRevision > 0 {
		return nil
	}
	b.lastSentRevision = b.lastRevision
	out := make([]byte, len(b.frameBuffer))
	copy(out, b.frameBuffer)
	return out
}

// GetWidth returns the canvas pixel width.
func (b *Bridge) GetWidth() int { return canvasWidth }

// GetHeight returns the canvas pixel height.
func (b *Bridge) GetHeight() int { return canvasHeight }

// SendTouchDown enqueues a pointer-pressed event at the given canvas coordinates.
func (b *Bridge) SendTouchDown(x, y int) {
	b.mu.Lock()
	b.lastSentRevision = 0
	b.mu.Unlock()
	select {
	case b.touchQueue <- touchEvent{x: x, y: y, pressed: true}:
	default:
		// Drop when queue is full — acceptable for rapid touch sequences.
	}
}

// SendTouchUp enqueues a pointer-released event at the given canvas coordinates.
func (b *Bridge) SendTouchUp(x, y int) {
	b.mu.Lock()
	b.lastSentRevision = 0
	b.mu.Unlock()
	select {
	case b.touchQueue <- touchEvent{x: x, y: y, pressed: false}:
	default:
	}
}

// SendTouchDrag enqueues a pointer-dragged event at the given canvas coordinates.
func (b *Bridge) SendTouchDrag(x, y int) {
	b.mu.Lock()
	b.lastSentRevision = 0
	b.mu.Unlock()
	select {
	case b.touchQueue <- touchEvent{x: x, y: y, dragged: true}:
	default:
	}
}

// SendKeyDown dispatches a keyPressed event with keyCode to the active canvas.
func (b *Bridge) SendKeyDown(keyCode int) {
	b.dispatchKey(keyCode, true)
}

// SendKeyUp dispatches a keyReleased event with keyCode to the active canvas.
func (b *Bridge) SendKeyUp(keyCode int) {
	b.dispatchKey(keyCode, false)
}

// SendKeyPress dispatches keyPressed and keyReleased with the given keyCode to the active canvas.
func (b *Bridge) SendKeyPress(keyCode int) {
	b.SendKeyDown(keyCode)
	time.AfterFunc(50*time.Millisecond, func() {
		b.SendKeyUp(keyCode)
	})
}

// SetShouldPaint records whether Flutter should rasterize the framebuffer.
// The VM tick loop always continues; GetFrameBuffer still returns the last copy.
func (b *Bridge) SetShouldPaint(paint bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.shouldPaint = paint
	if paint {
		b.lastSentRevision = 0
	}
}

// ShouldPaint reports the last SetShouldPaint value (default true).
func (b *Bridge) ShouldPaint() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.shouldPaint
}

// StartUpgrade unmarshals configJSON and starts the upgrade bot.
// Returns "" on success or a human-readable error (including "not connected").
func (b *Bridge) StartUpgrade(configJSON string) string {
	var cfg upgradebot.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return fmt.Sprintf("invalid config: %v", err)
	}
	b.mu.Lock()
	if b.vmCtx == nil || b.vmCtx.ActiveSocket == nil {
		b.mu.Unlock()
		return "not connected"
	}
	b.maybeWireTapLocked()
	bot := b.bot
	b.mu.Unlock()
	if bot == nil {
		return "not connected"
	}
	return bot.Start(cfg)
}

// StopUpgrade stops the upgrade bot if it has been created.
func (b *Bridge) StopUpgrade() {
	b.mu.Lock()
	bot := b.bot
	b.mu.Unlock()
	if bot != nil {
		bot.Stop()
	}
}

// GetUpgradeStatus returns the bot status JSON, or a zero-value object if idle.
func (b *Bridge) GetUpgradeStatus() string {
	b.mu.Lock()
	bot := b.bot
	b.mu.Unlock()
	if bot == nil {
		return emptyUpgradeStatusJSON
	}
	return bot.StatusJSON()
}

// Stop halts the background loop and blocks until the goroutine exits.
func (b *Bridge) Stop() {
	select {
	case <-b.stopCh:
		// already stopped
	default:
		close(b.stopCh)
	}
	b.wg.Wait()
}

// IsStarted returns true if LoadJAR completed successfully.
func (b *Bridge) IsStarted() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.started
}

// CopyFrontBufferInto copies the current front-buffer pixels into dst (draw.Image).
func (b *Bridge) CopyFrontBufferInto(dst draw.Image) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.gfx == nil || b.gfx.FrontBuffer == nil {
		return
	}
	draw.Draw(dst, dst.Bounds(), b.gfx.FrontBuffer, b.gfx.FrontBuffer.Bounds().Min, draw.Src)
}

type TextInputInfo struct {
	Active     bool   `json:"active"`
	Title      string `json:"title"`
	Text       string `json:"text"`
	MaxSize    int    `json:"max_size"`
	IsPassword bool   `json:"is_password"`
}

// GetTextInputInfoJSON returns information about any active TextBox waiting for user text input.
func (b *Bridge) GetTextInputInfoJSON() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.vmCtx == nil || b.vmCtx.PendingTextBox == nil {
		return `{"active":false}`
	}
	tb := b.vmCtx.PendingTextBox
	title := b.vmCtx.ExtractString(tb.GetField("title").Ref)
	text := b.vmCtx.ExtractString(tb.GetField("text").Ref)
	maxSize := int(tb.GetField("maxSize").Num)
	if maxSize <= 0 {
		maxSize = 200
	}
	constraints := int(tb.GetField("constraints").Num)
	isPassword := (constraints&0x10000 != 0) || (constraints&0xFFFF == 2) ||
		strings.Contains(strings.ToLower(title), "mật khẩu") ||
		strings.Contains(strings.ToLower(title), "pass")

	info := TextInputInfo{
		Active:     true,
		Title:      title,
		Text:       text,
		MaxSize:    maxSize,
		IsPassword: isPassword,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return `{"active":false}`
	}
	return string(data)
}

// SubmitTextInput updates the pending TextBox with user-entered text and fires its OK commandAction.
func (b *Bridge) SubmitTextInput(text string) {
	b.mu.Lock()
	if b.vmCtx == nil || b.vmCtx.PendingTextBox == nil {
		b.mu.Unlock()
		return
	}
	tb := b.vmCtx.PendingTextBox
	b.mu.Unlock()

	b.vmCtx.CallSerialized(func() {
		tb.SetField("text", vm.Slot{Ref: b.vmCtx.NewString(text)})

		listenerRef := tb.GetField("listener").Ref
		listenerObj, _ := listenerRef.(*vm.Object)
		if listenerObj != nil && listenerObj.Class != nil {
			okCmdRef := tb.GetField("okCommand").Ref
			if okCmdRef == nil {
				okCmdRef = b.vmCtx.NewCommand("OK", 4, 0)
			}
			cmdActionM := listenerObj.Class.GetMethod("commandAction", "(Ljavax/microedition/lcdui/Command;Ljavax/microedition/lcdui/Displayable;)V")
			if cmdActionM != nil {
				_, _ = b.vmCtx.InvokeMethod(listenerObj.Class, cmdActionM, []vm.Slot{
					{Ref: listenerObj},
					{Ref: okCmdRef},
					{Ref: tb},
				})
			}
		}

		b.mu.Lock()
		b.vmCtx.PendingTextBox = nil
		b.lastSentRevision = 0
		b.mu.Unlock()
	})
}

// CancelTextInput dismisses the pending TextBox and fires its Cancel commandAction.
func (b *Bridge) CancelTextInput() {
	b.mu.Lock()
	if b.vmCtx == nil || b.vmCtx.PendingTextBox == nil {
		b.mu.Unlock()
		return
	}
	tb := b.vmCtx.PendingTextBox
	b.mu.Unlock()

	b.vmCtx.CallSerialized(func() {
		listenerRef := tb.GetField("listener").Ref
		listenerObj, _ := listenerRef.(*vm.Object)
		if listenerObj != nil && listenerObj.Class != nil {
			cancelCmdRef := tb.GetField("cancelCommand").Ref
			if cancelCmdRef == nil {
				cancelCmdRef = b.vmCtx.NewCommand("Cancel", 3, 0)
			}
			cmdActionM := listenerObj.Class.GetMethod("commandAction", "(Ljavax/microedition/lcdui/Command;Ljavax/microedition/lcdui/Displayable;)V")
			if cmdActionM != nil {
				_, _ = b.vmCtx.InvokeMethod(listenerObj.Class, cmdActionM, []vm.Slot{
					{Ref: listenerObj},
					{Ref: cancelCmdRef},
					{Ref: tb},
				})
			}
		}

		b.mu.Lock()
		b.vmCtx.PendingTextBox = nil
		b.lastSentRevision = 0
		b.mu.Unlock()
	})
}

