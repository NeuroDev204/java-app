package vm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"java-app/pkg/midp"
)

// HTTPResult is the body of a MIDP HttpConnection opened via Connector.open("http://...").
type HTTPResult struct {
	Status int
	Body   []byte
}

func httpResultFromRef(ref interface{}) *HTTPResult {
	obj, ok := ref.(*Object)
	if !ok || obj == nil {
		return nil
	}
	hr, _ := obj.Extra.(*HTTPResult)
	return hr
}

func fetchHTTP(url string) (*HTTPResult, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return &HTTPResult{Status: resp.StatusCode, Body: body}, nil
}

func parseSocketURL(name string) (string, int, error) {
	const prefix = "socket://"
	if !strings.HasPrefix(name, prefix) {
		return "", 0, fmt.Errorf("unsupported connector url: %s", name)
	}
	rest := name[len(prefix):]
	host, portStr, ok := strings.Cut(rest, ":")
	if !ok || host == "" || portStr == "" {
		return "", 0, fmt.Errorf("invalid socket url: %s", name)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid socket port: %s", name)
	}
	return host, port, nil
}

func isSocketConnectionClass(clsName string) bool {
	return strings.Contains(clsName, "SocketConnection") ||
		strings.Contains(clsName, "OutputConnection") ||
		strings.Contains(clsName, "InputConnection") ||
		strings.Contains(clsName, "StreamConnection") ||
		clsName == "javax/microedition/io/Connection"
}

func socketTapFromRef(ref interface{}) *midp.SocketTap {
	obj, ok := ref.(*Object)
	if !ok || obj == nil {
		return nil
	}
	tap, _ := obj.Extra.(*midp.SocketTap)
	return tap
}

func (ctx *VMContext) closeSocketTap(tap *midp.SocketTap) {
	if tap == nil {
		return
	}
	_ = tap.Close()
	if ctx.ActiveSocket == tap {
		ctx.ActiveSocket = nil
	}
}

func (ctx *VMContext) handleConnectorOpen(mDesc string, frame *Frame) {
	params := mDesc
	if end := strings.Index(mDesc, ")"); end >= 0 {
		params = mDesc[:end]
	}
	if strings.Contains(params, "I") {
		frame.OperandStack.PopInt()
	}
	if strings.Contains(params, "Z") {
		frame.OperandStack.PopInt()
	}
	nameRef := frame.OperandStack.PopRef()
	name := extractString(nameRef)
	log.Printf("[CONNECTOR] open %s", name)
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		res, err := fetchHTTP(name)
		if err != nil {
			log.Printf("[CONNECTOR] http %s: %v", name, err)
			frame.OperandStack.PushRef(nil)
			return
		}
		log.Printf("[CONNECTOR] http %s status=%d bytes=%d", name, res.Status, len(res.Body))
		frame.OperandStack.PushRef(&Object{Extra: res})
		return
	}
	host, port, err := parseSocketURL(name)
	if err != nil {
		log.Printf("[CONNECTOR] unsupported url %s: %v", name, err)
		frame.OperandStack.PushRef(nil)
		return
	}
	if ctx.ActiveSocket != nil {
		// One TCP session at a time: drop the previous tap before Dial.
		ctx.closeSocketTap(ctx.ActiveSocket)
	}
	tap, err := midp.OpenSocket(host, port)
	if err != nil {
		log.Printf("[CONNECTOR] open %s: %v", name, err)
		frame.OperandStack.PushRef(nil)
		return
	}
	ctx.ActiveSocket = tap
	frame.OperandStack.PushRef(&Object{Extra: tap})
}

func (ctx *VMContext) handleHTTPNative(clsName, mName, mDesc string, frame *Frame, opcode byte) bool {
	paramCount := countMethodParamSlots(mDesc)
	var recv interface{}
	if opcode != OpInvokestatic && frame.OperandStack.size > uint(paramCount) {
		recv = frame.OperandStack.slots[frame.OperandStack.size-1-uint(paramCount)].Ref
	}
	hr := httpResultFromRef(recv)
	isHTTPCls := strings.Contains(clsName, "HttpConnection") || strings.Contains(clsName, "ContentConnection")
	if hr == nil && !isHTTPCls {
		return false
	}
	switch mName {
	case "setRequestMethod":
		_ = frame.OperandStack.PopRef()
		_ = frame.OperandStack.PopRef()
		return true
	case "setRequestProperty":
		_ = frame.OperandStack.PopRef()
		_ = frame.OperandStack.PopRef()
		_ = frame.OperandStack.PopRef()
		return true
	case "getResponseCode":
		_ = frame.OperandStack.PopRef()
		if hr != nil {
			frame.OperandStack.PushInt(int32(hr.Status))
		} else {
			frame.OperandStack.PushInt(0)
		}
		return true
	case "getLength":
		_ = frame.OperandStack.PopRef()
		n := 0
		if hr != nil {
			n = len(hr.Body)
		}
		frame.OperandStack.PushInt(int32(n))
		return true
	case "openInputStream", "openDataInputStream":
		_ = frame.OperandStack.PopRef()
		body := []byte{}
		if hr != nil {
			body = hr.Body
		}
		frame.OperandStack.PushRef(&Object{Extra: &DataStreamReader{Data: body, Pos: 0}})
		return true
	default:
		return false
	}
}

func (ctx *VMContext) handleSocketConnectionNative(mName, mDesc string, frame *Frame, opcode byte) bool {
	switch mName {
	case "openInputStream", "openOutputStream", "openDataInputStream", "openDataOutputStream":
		objRef := frame.OperandStack.PopRef()
		if tap := socketTapFromRef(objRef); tap != nil {
			frame.OperandStack.PushRef(&Object{Extra: tap})
			return true
		}
		frame.OperandStack.PushRef(nil)
		return true
	case "close":
		paramCount := countMethodParamSlots(mDesc)
		for i := 0; i < paramCount; i++ {
			frame.OperandStack.PopRef()
		}
		var objRef interface{}
		if opcode != OpInvokestatic {
			objRef = frame.OperandStack.PopRef()
		}
		ctx.closeSocketTap(socketTapFromRef(objRef))
		return true
	default:
		return false
	}
}

func readTapBytes(tap *midp.SocketTap, dst []byte) (int, error) {
	total := 0
	for total < len(dst) {
		n, err := tap.ReadPlain(dst[total:])
		if n > 0 {
			total += n
		}
		if err != nil {
			return total, err
		}
		if n == 0 {
			break
		}
	}
	return total, nil
}

func copyReadBytesToRef(dstRef interface{}, offset int, dst []byte, n int) {
	if n <= 0 {
		return
	}
	if obj, ok := dstRef.(*Object); ok && obj != nil {
		obj.Extra = dst
		return
	}
	if slots, ok := dstRef.([]Slot); ok {
		for i := 0; i < n; i++ {
			idx := offset + i
			if idx >= 0 && idx < len(slots) {
				slots[idx] = Slot{Num: int32(dst[idx])}
			}
		}
	}
}

func (ctx *VMContext) handleSocketInputRead(objRef, dstRef interface{}, offset, length int32, mName, mDesc string, frame *Frame) bool {
	tap := socketTapFromRef(objRef)
	if tap == nil {
		return false
	}
	dstBytes := extractBytes(dstRef)
	if dstBytes == nil {
		if hasReturnValue(mDesc) {
			frame.OperandStack.PushInt(-1)
		}
		return true
	}
	explicitRange := mName == "readFully" || strings.Contains(mDesc, "II")
	if !explicitRange {
		offset = 0
		length = int32(len(dstBytes))
	} else {
		if offset < 0 || length < 0 {
			if hasReturnValue(mDesc) {
				frame.OperandStack.PushInt(-1)
			}
			return true
		}
		if int(offset) > len(dstBytes) || int(offset)+int(length) > len(dstBytes) {
			if hasReturnValue(mDesc) {
				frame.OperandStack.PushInt(-1)
			}
			return true
		}
	}
	if length < 0 {
		length = 0
	}
	dst := dstBytes[offset : int(offset)+int(length)]
	var n int
	var err error
	if explicitRange {
		n, err = readTapBytes(tap, dst)
		if mName == "readFully" && (err != nil || n < len(dst)) {
			if frame != nil {
				frame.PendingException = ctx.NewException("java/io/EOFException", "EOF in readFully")
			}
			return true
		}
	} else if len(dst) > 0 {
		n, err = tap.ReadPlain(dst)
	}
	copyReadBytesToRef(dstRef, int(offset), dstBytes, n)
	if hasReturnValue(mDesc) {
		if n == 0 {
			frame.OperandStack.PushInt(-1)
		} else {
			frame.OperandStack.PushInt(int32(n))
		}
	}
	if n == 0 && err != nil {
		time.Sleep(20 * time.Millisecond)
	}
	return true
}

func (ctx *VMContext) pushTapInt(frame *Frame, tap *midp.SocketTap, size int, signed bool) {
	buf := make([]byte, size)
	n, err := readTapBytes(tap, buf)
	if err != nil || n < size {
		if frame != nil {
			frame.PendingException = ctx.NewException("java/io/EOFException", "EOF on socket")
		} else if size == 1 {
			frame.OperandStack.PushInt(-1)
		} else {
			frame.OperandStack.PushInt(0)
		}
		return
	}
	switch size {
	case 1:
		if signed {
			frame.OperandStack.PushInt(int32(int8(buf[0])))
		} else {
			frame.OperandStack.PushInt(int32(buf[0]))
		}
	case 2:
		v := binary.BigEndian.Uint16(buf)
		if signed {
			frame.OperandStack.PushInt(int32(int16(v)))
		} else {
			frame.OperandStack.PushInt(int32(v))
		}
	case 4:
		frame.OperandStack.PushInt(int32(binary.BigEndian.Uint32(buf)))
	default:
		frame.OperandStack.PushInt(0)
	}
}

func streamBufFromRef(ref interface{}) *bytes.Buffer {
	if ref == nil {
		return nil
	}
	if buf, ok := ref.(*bytes.Buffer); ok {
		return buf
	}
	obj, ok := ref.(*Object)
	if !ok || obj == nil {
		return nil
	}
	if buf, ok := obj.Extra.(*bytes.Buffer); ok {
		return buf
	}
	if inner, ok := obj.Extra.(*Object); ok {
		return streamBufFromRef(inner)
	}
	return nil
}

func (ctx *VMContext) writeStreamBytes(frame *Frame, objRef interface{}, p []byte) {
	if len(p) == 0 {
		return
	}
	if tap := socketTapFromRef(objRef); tap != nil {
		if _, err := tap.WritePlain(p); err != nil {
			if frame != nil {
				frame.PendingException = ctx.NewException("java/io/IOException", err.Error())
			}
		}
		return
	}
	if buf := streamBufFromRef(objRef); buf != nil {
		_, _ = buf.Write(p)
	}
}

func (ctx *VMContext) handleOutputStreamNative(mName, mDesc string, frame *Frame, opcode byte) bool {
	switch mName {
	case "<init>":
		paramCount := countMethodParamSlots(mDesc)
		var extra interface{}
		if paramCount > 0 {
			arg := frame.OperandStack.PopRef()
			if tap := socketTapFromRef(arg); tap != nil {
				extra = tap
			} else if buf := streamBufFromRef(arg); buf != nil {
				extra = buf
			} else {
				extra = new(bytes.Buffer)
			}
			for i := 1; i < paramCount; i++ {
				frame.OperandStack.PopSlot()
			}
		} else {
			extra = new(bytes.Buffer)
		}
		objRef := frame.OperandStack.PopRef()
		if obj, ok := objRef.(*Object); ok && obj != nil {
			obj.Extra = extra
		}
		return true
	case "write", "writeByte":
		if strings.HasPrefix(mDesc, "([BII)") {
			length := frame.OperandStack.PopInt()
			offset := frame.OperandStack.PopInt()
			srcRef := frame.OperandStack.PopRef()
			objRef := frame.OperandStack.PopRef()
			buf := extractBytes(srcRef)
			if buf == nil || offset < 0 || length < 0 {
				return true
			}
			end := int(offset) + int(length)
			if end > len(buf) {
				return true
			}
			ctx.writeStreamBytes(frame, objRef, buf[int(offset):end])
			return true
		}
		if strings.HasPrefix(mDesc, "([B)") {
			srcRef := frame.OperandStack.PopRef()
			objRef := frame.OperandStack.PopRef()
			src := extractBytes(srcRef)
			if src != nil {
				ctx.writeStreamBytes(frame, objRef, src)
			}
			return true
		}
		val := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		ctx.writeStreamBytes(frame, objRef, []byte{byte(val)})
		return true
	case "writeShort":
		val := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(val))
		ctx.writeStreamBytes(frame, objRef, buf[:])
		return true
	case "writeInt":
		val := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(val))
		ctx.writeStreamBytes(frame, objRef, buf[:])
		return true
	case "writeBoolean":
		val := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		b := byte(0)
		if val != 0 {
			b = 1
		}
		ctx.writeStreamBytes(frame, objRef, []byte{b})
		return true
	case "writeChar":
		val := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(val))
		ctx.writeStreamBytes(frame, objRef, buf[:])
		return true
	case "writeUTF":
		strRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		s := extractString(strRef)
		b := []byte(s)
		var lenBuf [2]byte
		binary.BigEndian.PutUint16(lenBuf[:], uint16(len(b)))
		ctx.writeStreamBytes(frame, objRef, lenBuf[:])
		ctx.writeStreamBytes(frame, objRef, b)
		return true
	case "close":
		objRef := frame.OperandStack.PopRef()
		ctx.closeSocketTap(socketTapFromRef(objRef))
		return true
	case "flush":
		if opcode != OpInvokestatic {
			frame.OperandStack.PopRef()
		}
		return true
	default:
		return false
	}
}

func (ctx *VMContext) handleByteArrayOutputStreamNative(mName, mDesc string, frame *Frame, opcode byte) bool {
	switch mName {
	case "<init>":
		if strings.HasPrefix(mDesc, "(I)") {
			_ = frame.OperandStack.PopInt()
		}
		objRef := frame.OperandStack.PopRef()
		if obj, ok := objRef.(*Object); ok && obj != nil {
			obj.Extra = new(bytes.Buffer)
		}
		return true
	case "write", "writeByte":
		if strings.HasPrefix(mDesc, "([BII)") {
			length := frame.OperandStack.PopInt()
			offset := frame.OperandStack.PopInt()
			srcRef := frame.OperandStack.PopRef()
			objRef := frame.OperandStack.PopRef()
			buf := streamBufFromRef(objRef)
			if buf == nil {
				return true
			}
			data := extractBytes(srcRef)
			if data == nil || offset < 0 || length < 0 {
				return true
			}
			end := int(offset) + int(length)
			if end > len(data) {
				return true
			}
			buf.Write(data[int(offset):end])
			return true
		}
		if strings.HasPrefix(mDesc, "([B)") {
			srcRef := frame.OperandStack.PopRef()
			objRef := frame.OperandStack.PopRef()
			buf := streamBufFromRef(objRef)
			if buf == nil {
				return true
			}
			data := extractBytes(srcRef)
			if data != nil {
				buf.Write(data)
			}
			return true
		}
		val := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		if buf := streamBufFromRef(objRef); buf != nil {
			buf.WriteByte(byte(val))
		}
		return true
	case "toByteArray":
		objRef := frame.OperandStack.PopRef()
		buf := streamBufFromRef(objRef)
		if buf != nil {
			data := make([]byte, buf.Len())
			copy(data, buf.Bytes())
			frame.OperandStack.PushRef(data)
		} else {
			frame.OperandStack.PushRef([]byte{})
		}
		return true
	case "reset":
		objRef := frame.OperandStack.PopRef()
		if buf := streamBufFromRef(objRef); buf != nil {
			buf.Reset()
		}
		return true
	case "size":
		objRef := frame.OperandStack.PopRef()
		if buf := streamBufFromRef(objRef); buf != nil {
			frame.OperandStack.PushInt(int32(buf.Len()))
		} else {
			frame.OperandStack.PushInt(0)
		}
		return true
	case "close":
		if opcode != OpInvokestatic {
			_ = frame.OperandStack.PopRef()
		}
		return true
	case "toString":
		if strings.HasPrefix(mDesc, "(Ljava/lang/String;)") {
			_ = frame.OperandStack.PopRef()
		}
		objRef := frame.OperandStack.PopRef()
		if buf := streamBufFromRef(objRef); buf != nil {
			frame.OperandStack.PushRef(buf.String())
		} else {
			frame.OperandStack.PushRef("")
		}
		return true
	case "writeTo":
		outRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		if buf := streamBufFromRef(objRef); buf != nil {
			ctx.writeStreamBytes(frame, outRef, buf.Bytes())
		}
		return true
	default:
		return false
	}
}
