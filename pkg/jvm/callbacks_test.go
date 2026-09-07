package jvm

import (
	"sync"
	"testing"
)

func TestRenderQueue(t *testing.T) {
	q := GetRenderQueue()
	q.Clear()

	if q.Len() != 0 {
		t.Fatalf("expected initial empty queue, got len %d", q.Len())
	}

	q.Push(RenderCommand{Type: CmdSetColor, RGB: 0x00FF00})
	q.Push(RenderCommand{Type: CmdFillRect, X: 10, Y: 20, Width: 100, Height: 50})

	if q.Len() != 2 {
		t.Fatalf("expected len 2, got %d", q.Len())
	}

	cmds := q.PopAll()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 popped commands, got %d", len(cmds))
	}

	if cmds[0].Type != CmdSetColor || cmds[0].RGB != 0x00FF00 {
		t.Errorf("unexpected first command: %+v", cmds[0])
	}

	if cmds[1].Type != CmdFillRect || cmds[1].X != 10 || cmds[1].Y != 20 || cmds[1].Width != 100 || cmds[1].Height != 50 {
		t.Errorf("unexpected second command: %+v", cmds[1])
	}

	if q.Len() != 0 {
		t.Errorf("expected empty queue after PopAll, got len %d", q.Len())
	}
}

func TestRenderQueueConcurrent(t *testing.T) {
	q := GetRenderQueue()
	q.Clear()

	const numGoroutines = 10
	const numPushesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numPushesPerGoroutine; j++ {
				q.Push(RenderCommand{
					Type:   CmdFillRect,
					X:      id,
					Y:      j,
					Width:  10,
					Height: 10,
				})
			}
		}(i)
	}

	wg.Wait()

	if q.Len() != numGoroutines*numPushesPerGoroutine {
		t.Errorf("expected %d total commands, got %d", numGoroutines*numPushesPerGoroutine, q.Len())
	}

	cmds := q.PopAll()
	if len(cmds) != numGoroutines*numPushesPerGoroutine {
		t.Errorf("expected %d popped commands, got %d", numGoroutines*numPushesPerGoroutine, len(cmds))
	}
}

func TestInvokeMainEndToEnd(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	q := GetRenderQueue()
	q.Clear()

	err = vm.InvokeMain("Test")
	if err != nil {
		t.Fatalf("InvokeMain(\"Test\") failed: %v", err)
	}

	cmds := q.PopAll()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands in RenderQueue after InvokeMain, got %d", len(cmds))
	}

	// Verify command 1: SetColor
	if cmds[0].Type != CmdSetColor || cmds[0].RGB != 0x00FF00 {
		t.Errorf("command 0 mismatched: %+v", cmds[0])
	}

	// Verify command 2: FillRect(10, 20, 100, 50)
	if cmds[1].Type != CmdFillRect || cmds[1].X != 10 || cmds[1].Y != 20 || cmds[1].Width != 100 || cmds[1].Height != 50 {
		t.Errorf("command 1 mismatched: %+v", cmds[1])
	}
}

func TestInvokeMainInvalidClass(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	err = vm.InvokeMain("NonExistentClass")
	if err == nil {
		t.Error("expected error when invoking main on non-existent class, got nil")
	}
}

func TestRecordStoreCallbacks(t *testing.T) {
	DeleteRecordStoreGo("test_store")
	defer DeleteRecordStoreGo("test_store")

	hnd := OpenRecordStoreGo("test_store", true)
	if hnd <= 0 {
		t.Fatalf("expected positive handle, got %d", hnd)
	}

	data := []byte("hello world")
	recId := AddRecordGo(hnd, data)
	if recId != 1 {
		t.Errorf("expected recId 1, got %d", recId)
	}

	num := GetNumRecordsGo(hnd)
	if num != 1 {
		t.Errorf("expected 1 record, got %d", num)
	}

	sz := GetRecordSizeGo(hnd, recId)
	if sz != int32(len(data)) {
		t.Errorf("expected record size %d, got %d", len(data), sz)
	}

	CloseRecordStoreGo(hnd)
}

func TestVRAMImageCallbacks(t *testing.T) {
	imgId := CreateImageWHGo(100, 200)
	if imgId <= 0 {
		t.Fatalf("expected positive image ID, got %d", imgId)
	}

	w := ImageGetWidthGo(imgId)
	h := ImageGetHeightGo(imgId)

	if w != 100 || h != 200 {
		t.Errorf("expected 100x200, got %dx%d", w, h)
	}
}
