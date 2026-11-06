package jvm

/*
#cgo CFLAGS: -I/usr/lib/jvm/java-21-openjdk-amd64/include -I/usr/lib/jvm/java-21-openjdk-amd64/include/linux
#include <jni.h>
#include <stdlib.h>
#include <string.h>

jint register_graphics_natives(JavaVM *vm, JNIEnv *env);
jint safe_invoke_main(JavaVM *vm, JNIEnv *env, const char *class_name);
jint safe_invoke_main_args(JavaVM *vm, JNIEnv *env, const char *class_name, const char *arg0);
jobject c_new_global_ref(JNIEnv *env, jobject obj);
void c_delete_global_ref(JNIEnv *env, jobject obj);
void c_dispatch_key_event(JavaVM *vm, JNIEnv *env, jobject canvas, jint key_code, jboolean pressed);
void c_dispatch_key_repeated(JavaVM *vm, JNIEnv *env, jobject canvas, jint key_code);
void c_dispatch_pointer_event(JavaVM *vm, JNIEnv *env, jobject canvas, jint x, jint y, jint event_type);
void c_print_class_name(JNIEnv *env, jobject obj);
int c_is_textbox(JNIEnv *env, jobject obj);
void c_handle_textbox_safe(JavaVM *vm, jobject obj, const char *new_text);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"java-app/pkg/midp"
)

func ensureRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

// CommandType defines the type of render command queued from JNI callbacks.
type CommandType int

const (
	CmdFillRect CommandType = iota
	CmdDrawRect
	CmdDrawLine
	CmdSetColor
	CmdDrawString
	CmdDrawImage
	CmdDrawRegion
	CmdSetClip
)

// RenderCommand encapsulates parameters for a single render operation.
type RenderCommand struct {
	Type   CommandType
	X, Y   int
	Width  int
	Height int
	RGB    int
	Str    string
	ImgID  int32
}

// RenderQueue provides a thread-safe FIFO queue for JNI render callbacks.
type RenderQueue struct {
	mu       sync.Mutex
	commands []RenderCommand
}

var (
	globalRenderQueue *RenderQueue
	renderQueueOnce   sync.Once

	// Active Graphics target
	activeGraphics      *midp.Graphics
	activeGraphicsMutex sync.RWMutex

	// VRAM Images
	vramImages   = make(map[int32]*image.RGBA)
	nextImageID  int32 = 1
	vramMutex    sync.RWMutex

	// RecordStores
	recordStores  = make(map[string]map[int][]byte)
	storeHandles  = make(map[int32]string)
	nextStoreHnd  int32 = 1
	rmsMutex      sync.RWMutex

	// Active Canvas
	currentCanvasRef C.jobject
	canvasMutex      sync.Mutex
)

// GetRenderQueue returns the thread-safe global RenderQueue instance.
func GetRenderQueue() *RenderQueue {
	renderQueueOnce.Do(func() {
		globalRenderQueue = &RenderQueue{
			commands: make([]RenderCommand, 0),
		}
	})
	return globalRenderQueue
}

func (q *RenderQueue) Push(cmd RenderCommand) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.commands) < 50000 {
		q.commands = append(q.commands, cmd)
	}
}

func (q *RenderQueue) PopAll() []RenderCommand {
	q.mu.Lock()
	defer q.mu.Unlock()
	cmds := q.commands
	q.commands = make([]RenderCommand, 0)
	return cmds
}

func (q *RenderQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.commands)
}

func (q *RenderQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.commands = make([]RenderCommand, 0)
}

func SetActiveGraphics(g *midp.Graphics) {
	activeGraphicsMutex.Lock()
	defer activeGraphicsMutex.Unlock()
	activeGraphics = g
	if globalDisplayGraphics == nil {
		globalDisplayGraphics = g
	}
}

func GetActiveGraphics() *midp.Graphics {
	activeGraphicsMutex.RLock()
	defer activeGraphicsMutex.RUnlock()
	return activeGraphics
}

// Forward key press/release event to current active Java Canvas
func DispatchKeyEvent(keyCode int, pressed bool) {
	canvasMutex.Lock()
	ref := currentCanvasRef
	canvasMutex.Unlock()

	if unsafe.Pointer(ref) == nil || globalJVM == nil {
		return
	}
	bPressed := C.jboolean(C.JNI_FALSE)
	if pressed {
		bPressed = C.jboolean(C.JNI_TRUE)
	}
	C.c_dispatch_key_event(globalJVM.vm, globalJVM.env, ref, C.jint(keyCode), bPressed)
}

func DispatchKeyRepeated(keyCode int) {
	canvasMutex.Lock()
	ref := currentCanvasRef
	canvasMutex.Unlock()

	if unsafe.Pointer(ref) == nil || globalJVM == nil {
		return
	}
	C.c_dispatch_key_repeated(globalJVM.vm, globalJVM.env, ref, C.jint(keyCode))
}

// Forward pointer event to current active Java Canvas
func DispatchPointerEvent(x, y int, eventType int) {
	canvasMutex.Lock()
	ref := currentCanvasRef
	canvasMutex.Unlock()

	if unsafe.Pointer(ref) == nil || globalJVM == nil {
		return
	}
	C.c_dispatch_pointer_event(globalJVM.vm, globalJVM.env, ref, C.jint(x), C.jint(y), C.jint(eventType))
}

// --- JNI Callbacks ---

var (
	vramGraphics      = make(map[int32]*midp.Graphics)
	vramGraphicsMutex sync.RWMutex
	globalDisplayGraphics *midp.Graphics
)

//export go_setActiveTarget
func go_setActiveTarget(targetId C.jint) {
	id := int32(targetId)
	if id == 0 {
		SetActiveGraphics(globalDisplayGraphics)
	} else {
		vramGraphicsMutex.RLock()
		g := vramGraphics[id]
		vramGraphicsMutex.RUnlock()
		if g == nil {
			vramMutex.RLock()
			img := vramImages[id]
			vramMutex.RUnlock()
			if img != nil {
				g = &midp.Graphics{
					Width:  img.Bounds().Dx(),
					Height: img.Bounds().Dy(),
					Buffer: img,
				}
				vramGraphicsMutex.Lock()
				vramGraphics[id] = g
				vramGraphicsMutex.Unlock()
			}
		}
		if g != nil {
			SetActiveGraphics(g)
		}
	}
}

//export go_fillRect
func go_fillRect(x, y, width, height C.jint) {
	cmd := RenderCommand{
		Type:   CmdFillRect,
		X:      int(x),
		Y:      int(y),
		Width:  int(width),
		Height: int(height),
	}
	GetRenderQueue().Push(cmd)
	if g := GetActiveGraphics(); g != nil {
		g.FillRect(int(x), int(y), int(width), int(height))
	}
}

//export go_drawRect
func go_drawRect(x, y, width, height C.jint) {
	cmd := RenderCommand{
		Type:   CmdDrawRect,
		X:      int(x),
		Y:      int(y),
		Width:  int(width),
		Height: int(height),
	}
	GetRenderQueue().Push(cmd)
	if g := GetActiveGraphics(); g != nil {
		g.DrawRect(int(x), int(y), int(width), int(height))
	}
}

//export go_drawLine
func go_drawLine(x1, y1, x2, y2 C.jint) {
	cmd := RenderCommand{
		Type:   CmdDrawLine,
		X:      int(x1),
		Y:      int(y1),
		Width:  int(x2),
		Height: int(y2),
	}
	GetRenderQueue().Push(cmd)
	if g := GetActiveGraphics(); g != nil {
		g.DrawLine(int(x1), int(y1), int(x2), int(y2))
	}
}

//export go_setColor
func go_setColor(rgb C.jint) {
	cmd := RenderCommand{
		Type: CmdSetColor,
		RGB:  int(rgb),
	}
	GetRenderQueue().Push(cmd)
	if g := GetActiveGraphics(); g != nil {
		r := int((rgb >> 16) & 0xFF)
		gVal := int((rgb >> 8) & 0xFF)
		b := int(rgb & 0xFF)
		g.SetColor(r, gVal, b)
	}
}

//export go_setColor3
func go_setColor3(r, g, b C.jint) {
	rgb := (int(r) << 16) | (int(g) << 8) | int(b)
	cmd := RenderCommand{
		Type: CmdSetColor,
		RGB:  rgb,
	}
	GetRenderQueue().Push(cmd)
	if gfx := GetActiveGraphics(); gfx != nil {
		gfx.SetColor(int(r), int(g), int(b))
	}
}

//export go_drawString
func go_drawString(str *C.char, x, y, anchor C.jint) {
	s := C.GoString(str)
	cmd := RenderCommand{
		Type: CmdDrawString,
		X:    int(x),
		Y:    int(y),
		Str:  s,
	}
	GetRenderQueue().Push(cmd)
	if g := GetActiveGraphics(); g != nil {
		g.DrawString(s, int(x), int(y), int(anchor))
	}
}

//export go_drawSubstring
func go_drawSubstring(str *C.char, offset, lenVal, x, y, anchor C.jint) {
	s := C.GoString(str)
	runes := []rune(s)
	off := int(offset)
	l := int(lenVal)
	if off >= 0 && off+l <= len(runes) {
		sub := string(runes[off : off+l])
		go_drawString(str, x, y, anchor)
		_ = sub
	}
}

//export go_drawChar
func go_drawChar(ch C.jchar, x, y, anchor C.jint) {
	s := string(rune(ch))
	if g := GetActiveGraphics(); g != nil {
		g.DrawString(s, int(x), int(y), int(anchor))
	}
}

//export go_drawChars
func go_drawChars(chars *C.jchar, offset, lenVal, x, y, anchor C.jint) {
	l := int(lenVal)
	off := int(offset)
	if chars == nil || l <= 0 {
		return
	}
	slice := (*[1 << 28]C.jchar)(unsafe.Pointer(chars))[off : off+l]
	runes := make([]rune, l)
	for i, c := range slice {
		runes[i] = rune(c)
	}
	s := string(runes)
	if g := GetActiveGraphics(); g != nil {
		g.DrawString(s, int(x), int(y), int(anchor))
	}
}

//export go_drawImage
func go_drawImage(imgId, x, y, anchor C.jint) {
	id := int32(imgId)
	cmd := RenderCommand{
		Type:  CmdDrawImage,
		X:     int(x),
		Y:     int(y),
		ImgID: id,
	}
	GetRenderQueue().Push(cmd)

	vramMutex.RLock()
	img := vramImages[id]
	vramMutex.RUnlock()

	if img != nil {
		if g := GetActiveGraphics(); g != nil {
			g.DrawImage(img, int(x), int(y), int(anchor))
		}
	}
}

//export go_drawRegion
func go_drawRegion(imgId, xSrc, ySrc, w, h, trans, xDst, yDst, anchor C.jint) {
	id := int32(imgId)
	cmd := RenderCommand{
		Type:   CmdDrawRegion,
		X:      int(xDst),
		Y:      int(yDst),
		Width:  int(w),
		Height: int(h),
		ImgID:  id,
	}
	GetRenderQueue().Push(cmd)

	vramMutex.RLock()
	img := vramImages[id]
	vramMutex.RUnlock()

	if img != nil {
		if g := GetActiveGraphics(); g != nil {
			g.DrawRegion(img, int(xSrc), int(ySrc), int(w), int(h), int(trans), int(xDst), int(yDst), int(anchor))
		}
	}
}

//export go_drawRGB
func go_drawRGB(rgbData *C.jint, dataLen C.jint, offset, scanlength, x, y, width, height C.jint, processAlpha C.jboolean) {
	if g := GetActiveGraphics(); g != nil {
		off := int(offset)
		scan := int(scanlength)
		w := int(width)
		h := int(height)
		xx := int(x)
		yy := int(y)
		pAlpha := processAlpha == C.JNI_TRUE

		slice := unsafe.Slice((*int32)(unsafe.Pointer(rgbData)), int(dataLen))
		g.DrawRGB(slice, off, scan, xx, yy, w, h, pAlpha)
	}
}

//export go_setClip
func go_setClip(x, y, w, h C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.SetClip(int(x), int(y), int(w), int(h))
	}
}

//export go_clipRect
func go_clipRect(x, y, w, h C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.ClipRect(int(x), int(y), int(w), int(h))
	}
}

//export go_getClipX
func go_getClipX() C.jint {
	if g := GetActiveGraphics(); g != nil {
		return C.jint(g.ClipX)
	}
	return 0
}

//export go_getClipY
func go_getClipY() C.jint {
	if g := GetActiveGraphics(); g != nil {
		return C.jint(g.ClipY)
	}
	return 0
}

//export go_getClipWidth
func go_getClipWidth() C.jint {
	if g := GetActiveGraphics(); g != nil {
		return C.jint(g.ClipW)
	}
	return 240
}

//export go_getClipHeight
func go_getClipHeight() C.jint {
	if g := GetActiveGraphics(); g != nil {
		return C.jint(g.ClipH)
	}
	return 320
}

//export go_translate
func go_translate(x, y C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.Translate(int(x), int(y))
	}
}

//export go_getTranslateX
func go_getTranslateX() C.jint {
	if g := GetActiveGraphics(); g != nil {
		return C.jint(g.TransX)
	}
	return 0
}

//export go_getTranslateY
func go_getTranslateY() C.jint {
	if g := GetActiveGraphics(); g != nil {
		return C.jint(g.TransY)
	}
	return 0
}

//export go_fillArc
func go_fillArc(x, y, w, h, start, arc C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.FillRect(int(x), int(y), int(w), int(h))
	}
}

//export go_drawArc
func go_drawArc(x, y, w, h, start, arc C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.DrawRect(int(x), int(y), int(w), int(h))
	}
}

//export go_fillRoundRect
func go_fillRoundRect(x, y, w, h, arcW, arcH C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.FillRect(int(x), int(y), int(w), int(h))
	}
}

//export go_drawRoundRect
func go_drawRoundRect(x, y, w, h, arcW, arcH C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.DrawRect(int(x), int(y), int(w), int(h))
	}
}

//export go_fillTriangle
func go_fillTriangle(x1, y1, x2, y2, x3, y3 C.jint) {
	if g := GetActiveGraphics(); g != nil {
		g.DrawLine(int(x1), int(y1), int(x2), int(y2))
		g.DrawLine(int(x2), int(y2), int(x3), int(y3))
		g.DrawLine(int(x3), int(y3), int(x1), int(y1))
	}
}

//export go_getFont
func go_getFont() {
}

//export go_setFont
func go_setFont() {
}


// --- Image Callbacks ---

//export go_createImageString
func go_createImageString(path *C.char) C.jint {
	p := C.GoString(path)
	data, err := os.ReadFile(p)
	if err != nil {
		// try relative clean
		cleanP := filepath.Clean(p)
		if stringsHasPrefix(cleanP, "/") {
			cleanP = cleanP[1:]
		}
		data, err = os.ReadFile(cleanP)
	}
	if err != nil || len(data) == 0 {
		// create a fallback 16x16 image
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		id := atomic.AddInt32(&nextImageID, 1)
		vramMutex.Lock()
		vramImages[id] = img
		vramMutex.Unlock()
		return C.jint(id)
	}
	img, err := midp.DecodeImage(data)
	var rgbaImg *image.RGBA
	if err != nil {
		rgbaImg = image.NewRGBA(image.Rect(0, 0, 16, 16))
	} else {
		rgbaImg = ensureRGBA(img)
	}
	id := atomic.AddInt32(&nextImageID, 1)
	vramMutex.Lock()
	vramImages[id] = rgbaImg
	vramMutex.Unlock()
	return C.jint(id)
}

//export go_createImageWH
func go_createImageWH(w, h C.jint) C.jint {
	width := int(w)
	height := int(h)
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	id := atomic.AddInt32(&nextImageID, 1)
	vramMutex.Lock()
	vramImages[id] = img
	vramMutex.Unlock()
	return C.jint(id)
}

//export go_createImageBytes
func go_createImageBytes(dataPtr *C.char, offset C.jint, length C.jint) C.jint {
	bytes := C.GoBytes(unsafe.Pointer(dataPtr), length)
	img, err := midp.DecodeImage(bytes)
	var rgbaImg *image.RGBA
	if err != nil {
		rgbaImg = image.NewRGBA(image.Rect(0, 0, 16, 16))
	} else {
		rgbaImg = ensureRGBA(img)
	}
	id := atomic.AddInt32(&nextImageID, 1)
	vramMutex.Lock()
	vramImages[id] = rgbaImg
	vramMutex.Unlock()
	return C.jint(id)
}

//export go_createImageRegion
func go_createImageRegion(srcId, x, y, w, h, trans C.jint) C.jint {
	idSrc := int32(srcId)
	vramMutex.RLock()
	srcImg := vramImages[idSrc]
	vramMutex.RUnlock()

	outW, outH := int(w), int(h)
	if outW <= 0 {
		outW = 1
	}
	if outH <= 0 {
		outH = 1
	}
	dstImg := image.NewRGBA(image.Rect(0, 0, outW, outH))

	if srcImg != nil {
		draw.Draw(dstImg, image.Rect(0, 0, outW, outH), srcImg, image.Pt(int(x), int(y)), draw.Src)
	}

	id := atomic.AddInt32(&nextImageID, 1)
	vramMutex.Lock()
	vramImages[id] = dstImg
	vramMutex.Unlock()
	return C.jint(id)
}

//export go_createRGBImage
func go_createRGBImage(rgbData *C.jint, dataLen C.jint, width, height C.jint, processAlpha C.jboolean) C.jint {
	w := int(width)
	h := int(height)
	if w <= 0 || h <= 0 {
		return 0
	}
	pAlpha := processAlpha == C.JNI_TRUE
	slice := unsafe.Slice((*int32)(unsafe.Pointer(rgbData)), int(dataLen))

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if idx >= len(slice) {
				continue
			}
			pixel := uint32(slice[idx])
			a := uint8(255)
			if pAlpha {
				a = uint8(pixel >> 24)
			}
			r := uint8((pixel >> 16) & 0xFF)
			g := uint8((pixel >> 8) & 0xFF)
			b := uint8(pixel & 0xFF)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	vramMutex.Lock()
	id := nextImageID
	nextImageID++
	vramImages[id] = img
	vramMutex.Unlock()
	return C.jint(id)
}

//export go_getRGB
func go_getRGB(imgId C.jint, rgbData *C.jint, dataLen C.jint, offset, scanlength, x, y, width, height C.jint) {
	vramMutex.RLock()
	img := vramImages[int32(imgId)]
	vramMutex.RUnlock()
	if img == nil {
		return
	}
	
	off := int(offset)
	scan := int(scanlength)
	w := int(width)
	h := int(height)
	xx := int(x)
	yy := int(y)
	
	slice := unsafe.Slice((*int32)(unsafe.Pointer(rgbData)), int(dataLen))
	
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			c := img.RGBAAt(xx+dx, yy+dy)
			pixel := (int32(c.A) << 24) | (int32(c.R) << 16) | (int32(c.G) << 8) | int32(c.B)
			idx := off + dy*scan + dx
			if idx < len(slice) {
				slice[idx] = pixel
			}
		}
	}
}

//export go_imageGetWidth
func go_imageGetWidth(imgId C.jint) C.jint {
	id := int32(imgId)
	vramMutex.RLock()
	img := vramImages[id]
	vramMutex.RUnlock()
	if img != nil {
		return C.jint(img.Bounds().Dx())
	}
	return 0
}

//export go_imageGetHeight
func go_imageGetHeight(imgId C.jint) C.jint {
	id := int32(imgId)
	vramMutex.RLock()
	img := vramImages[id]
	vramMutex.RUnlock()
	if img != nil {
		return C.jint(img.Bounds().Dy())
	}
	return 0
}

//export go_destroyImage
func go_destroyImage(imgId C.jint) {
	id := int32(imgId)
	vramMutex.Lock()
	delete(vramImages, id)
	vramMutex.Unlock()
}

func GetVramCount() int {
	vramMutex.RLock()
	defer vramMutex.RUnlock()
	return len(vramImages)
}

// --- Canvas Callbacks ---

//export go_serviceRepaints
func go_serviceRepaints() {
	// Repaint requested from Java
}

//export go_flushGraphics
func go_flushGraphics() {
	if globalDisplayGraphics != nil {
		globalDisplayGraphics.Flush()
	}
}

// --- Display Callbacks ---

var (
	textInputActive bool
	inputText       string
	targetTextBox   C.jobject
	textInputMutex  sync.Mutex
)

//export go_setCurrentDisplayable
func go_setCurrentDisplayable(env *C.JNIEnv, d C.jobject) {
	canvasMutex.Lock()
	defer canvasMutex.Unlock()
	if unsafe.Pointer(currentCanvasRef) != nil && env != nil {
		C.c_delete_global_ref(env, currentCanvasRef)
		currentCanvasRef = (C.jobject)(unsafe.Pointer(uintptr(0)))
	}
	if unsafe.Pointer(d) != nil && env != nil {
		currentCanvasRef = C.c_new_global_ref(env, d)
		
		if C.c_is_textbox(env, d) != 0 {
			textInputMutex.Lock()
			textInputActive = true
			inputText = ""
			targetTextBox = C.c_new_global_ref(env, d)
			textInputMutex.Unlock()
		} else {
			textInputMutex.Lock()
			textInputActive = false
			textInputMutex.Unlock()
		}
	}
}

func IsTextInputActive() bool {
	textInputMutex.Lock()
	defer textInputMutex.Unlock()
	return textInputActive
}

func GetInputText() string {
	textInputMutex.Lock()
	defer textInputMutex.Unlock()
	return inputText
}

func ProcessTextInput(chars []rune, backspace, enter bool) {
	textInputMutex.Lock()
	
	if !textInputActive {
		textInputMutex.Unlock()
		return
	}
	
	for _, c := range chars {
		if c >= 32 && c <= 126 {
			inputText += string(c)
		}
	}
	
	if backspace && len(inputText) > 0 {
		inputText = inputText[:len(inputText)-1]
	}
	
	var shouldHandleTextBox bool
	var textToHandle string
	var textBoxRef C.jobject

	if enter {
		textInputActive = false
		if unsafe.Pointer(targetTextBox) != nil && globalJVM != nil {
			shouldHandleTextBox = true
			textToHandle = inputText
			textBoxRef = targetTextBox
			targetTextBox = (C.jobject)(unsafe.Pointer(uintptr(0)))
		}
	}
	textInputMutex.Unlock()

	// Call JNI without holding the Go mutex to avoid reentrancy deadlock!
	if shouldHandleTextBox {
		cText := C.CString(textToHandle)
		C.c_handle_textbox_safe(globalJVM.vm, textBoxRef, cText)
		C.free(unsafe.Pointer(cText))
	}
}

//export go_vibrate
func go_vibrate(duration C.jint) {}

//export go_flashBacklight
func go_flashBacklight(duration C.jint) C.jboolean {
	return C.jboolean(C.JNI_TRUE)
}

// --- RecordStore Callbacks ---

const rmsFile = "rms_data.json"

type RMSRecord struct {
	ID   int    `json:"id"`
	Data []byte `json:"data"`
}

type RMSStore struct {
	Name    string      `json:"name"`
	Records []RMSRecord `json:"records"`
}

func init() {
	loadRMS()
}

func loadRMS() {
	data, err := os.ReadFile(rmsFile)
	if err != nil {
		return
	}
	var stores []RMSStore
	if err := json.Unmarshal(data, &stores); err != nil {
		return
	}
	rmsMutex.Lock()
	defer rmsMutex.Unlock()
	for _, store := range stores {
		m := make(map[int][]byte)
		for _, rec := range store.Records {
			m[rec.ID] = rec.Data
		}
		recordStores[store.Name] = m
	}
}

func saveRMS() {
	rmsMutex.RLock()
	var stores []RMSStore
	for name, m := range recordStores {
		var records []RMSRecord
		for id, data := range m {
			records = append(records, RMSRecord{ID: id, Data: data})
		}
		stores = append(stores, RMSStore{Name: name, Records: records})
	}
	rmsMutex.RUnlock()
	
	data, err := json.MarshalIndent(stores, "", "  ")
	if err == nil {
		os.WriteFile(rmsFile, data, 0644)
	}
}

//export go_openRecordStore
func go_openRecordStore(name *C.char, create C.jboolean) C.jint {
	storeName := C.GoString(name)
	rmsMutex.Lock()
	defer rmsMutex.Unlock()

	store, exists := recordStores[storeName]
	if !exists {
		if create == C.JNI_FALSE {
			return 0
		}
		store = make(map[int][]byte)
		recordStores[storeName] = store
	}

	hnd := atomic.AddInt32(&nextStoreHnd, 1)
	storeHandles[hnd] = storeName
	return C.jint(hnd)
}

//export go_closeRecordStore
func go_closeRecordStore(handle C.jint) {
	rmsMutex.Lock()
	defer rmsMutex.Unlock()
	delete(storeHandles, int32(handle))
}

//export go_deleteRecordStore
func go_deleteRecordStore(name *C.char) {
	storeName := C.GoString(name)
	rmsMutex.Lock()
	delete(recordStores, storeName)
	rmsMutex.Unlock()
	saveRMS()
}

//export go_listRecordStoresCount
func go_listRecordStoresCount() C.jint {
	rmsMutex.RLock()
	defer rmsMutex.RUnlock()
	return C.jint(len(recordStores))
}

//export go_listRecordStoreName
func go_listRecordStoreName(index C.jint) *C.char {
	rmsMutex.RLock()
	defer rmsMutex.RUnlock()
	idx := int(index)
	i := 0
	for name := range recordStores {
		if i == idx {
			return C.CString(name)
		}
		i++
	}
	return nil
}

//export go_getNumRecords
func go_getNumRecords(handle C.jint) C.jint {
	rmsMutex.RLock()
	defer rmsMutex.RUnlock()
	name, ok := storeHandles[int32(handle)]
	if !ok {
		return 0
	}
	store := recordStores[name]
	if store == nil {
		return 0
	}
	return C.jint(len(store))
}

//export go_getRecordSize
func go_getRecordSize(handle C.jint, recId C.jint) C.jint {
	rmsMutex.RLock()
	defer rmsMutex.RUnlock()
	name, ok := storeHandles[int32(handle)]
	if !ok {
		return -1
	}
	store := recordStores[name]
	if store == nil {
		return -1
	}
	data, ok := store[int(recId)]
	if !ok {
		return -1
	}
	return C.jint(len(data))
}

//export go_getRecordData
func go_getRecordData(handle C.jint, recId C.jint, outBuf *C.char) {
	rmsMutex.RLock()
	defer rmsMutex.RUnlock()
	name, ok := storeHandles[int32(handle)]
	if !ok {
		return
	}
	store := recordStores[name]
	if store == nil {
		return
	}
	data, ok := store[int(recId)]
	if !ok || len(data) == 0 {
		return
	}
	C.memcpy(unsafe.Pointer(outBuf), unsafe.Pointer(&data[0]), C.size_t(len(data)))
}

//export go_addRecord
func go_addRecord(handle C.jint, bytes *C.char, offset, lenVal C.jint) C.jint {
	rmsMutex.Lock()
	name, ok := storeHandles[int32(handle)]
	if !ok {
		rmsMutex.Unlock()
		return -1
	}
	store := recordStores[name]
	if store == nil {
		store = make(map[int][]byte)
		recordStores[name] = store
	}
	recId := len(store) + 1
	off := int(offset)
	l := int(lenVal)
	var recData []byte
	if bytes != nil && l > 0 {
		slice := C.GoBytes(unsafe.Pointer(bytes), C.int(off+l))
		recData = make([]byte, l)
		copy(recData, slice[off:off+l])
	} else {
		recData = make([]byte, 0)
	}
	store[recId] = recData
	rmsMutex.Unlock()
	saveRMS()
	return C.jint(recId)
}

//export go_setRecord
func go_setRecord(handle C.jint, recId C.jint, bytes *C.char, offset, lenVal C.jint) {
	rmsMutex.Lock()
	name, ok := storeHandles[int32(handle)]
	if !ok {
		rmsMutex.Unlock()
		return
	}
	store := recordStores[name]
	if store == nil {
		store = make(map[int][]byte)
		recordStores[name] = store
	}
	off := int(offset)
	l := int(lenVal)
	var recData []byte
	if bytes != nil && l > 0 {
		slice := C.GoBytes(unsafe.Pointer(bytes), C.int(off+l))
		recData = make([]byte, l)
		copy(recData, slice[off:off+l])
	} else {
		recData = make([]byte, 0)
	}
	store[int(recId)] = recData
	rmsMutex.Unlock()
	saveRMS()
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// RegisterNatives registers Java native methods with CGO handlers.
func (j *JVM) RegisterNatives() error {
	if j == nil || j.vm == nil || j.env == nil {
		return fmt.Errorf("JVM not initialized")
	}

	res := C.register_graphics_natives(j.vm, j.env)
	if res != 0 {
		return fmt.Errorf("register_graphics_natives failed with code %d", int(res))
	}
	return nil
}

// InvokeMain executes the static main method of the specified Java class.
func (j *JVM) InvokeMain(className string) error {
	return j.InvokeMainWithArgs(className, nil)
}

// InvokeMainWithArgs executes the static main method of the specified Java class with string arguments.
func (j *JVM) InvokeMainWithArgs(className string, args []string) error {
	if j == nil || j.vm == nil || j.env == nil {
		return fmt.Errorf("JVM not initialized")
	}

	cName := C.CString(className)
	defer C.free(unsafe.Pointer(cName))

	var cArg0 *C.char
	if len(args) > 0 {
		cArg0 = C.CString(args[0])
		defer C.free(unsafe.Pointer(cArg0))
	}

	res := C.safe_invoke_main_args(j.vm, j.env, cName, cArg0)
	if res != 0 {
		return fmt.Errorf("failed to invoke main for class %s (error code %d)", className, int(res))
	}

	return nil
}

func OpenRecordStoreGo(name string, create bool) int32 {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	bCreate := C.jboolean(C.JNI_FALSE)
	if create {
		bCreate = C.jboolean(C.JNI_TRUE)
	}
	return int32(go_openRecordStore(cName, bCreate))
}

func AddRecordGo(handle int32, data []byte) int32 {
	var cData *C.char
	if len(data) > 0 {
		cData = (*C.char)(unsafe.Pointer(&data[0]))
	}
	return int32(go_addRecord(C.jint(handle), cData, 0, C.jint(len(data))))
}

func GetNumRecordsGo(handle int32) int32 {
	return int32(go_getNumRecords(C.jint(handle)))
}

func GetRecordSizeGo(handle int32, recId int32) int32 {
	return int32(go_getRecordSize(C.jint(handle), C.jint(recId)))
}

func CloseRecordStoreGo(handle int32) {
	go_closeRecordStore(C.jint(handle))
}

func DeleteRecordStoreGo(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	go_deleteRecordStore(cName)
}

func CreateImageWHGo(w, h int) int32 {
	return int32(go_createImageWH(C.jint(w), C.jint(h)))
}

func ImageGetWidthGo(id int32) int {
	return int(go_imageGetWidth(C.jint(id)))
}

func ImageGetHeightGo(id int32) int {
	return int(go_imageGetHeight(C.jint(id)))
}
