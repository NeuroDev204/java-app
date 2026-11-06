package vm

import (
	"bytes"
	"path/filepath"
	"testing"
)

// A record added during play must still be there after a restart, otherwise the
// game re-downloads every item icon on each launch and the inventory sits on the
// loading spinner.
func TestRMSPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rms.json")
	SetRMSPath(path)
	t.Cleanup(func() { SetRMSPath("") })

	icon := []byte{0x89, 'P', 'N', 'G', 1, 2, 3}

	first := &VMContext{RecordStores: map[string]*RecordStoreData{}}
	first.loadDefaultRMS()
	first.RecordStores["image9999"] = &RecordStoreData{Records: [][]byte{icon}}
	first.saveRMS()

	second := &VMContext{RecordStores: map[string]*RecordStoreData{}}
	second.loadDefaultRMS()

	store, ok := second.RecordStores["image9999"]
	if !ok || store == nil {
		t.Fatal("downloaded record store did not survive restart")
	}
	if len(store.Records) != 1 || !bytes.Equal(store.Records[0], icon) {
		t.Fatalf("record round-trip = %v, want %v", store.Records, [][]byte{icon})
	}
	// Dynamic image snapshots are intentionally not restored as authoritative
	// icons; the server must provide the current image for those IDs.
	if _, ok := second.RecordStores["image4000"]; ok {
		t.Fatal("stale dynamic baseline image4000 should not be restored")
	}
}

// With no path configured nothing is written and nothing blows up.
func TestRMSSaveIsNoopWithoutPath(t *testing.T) {
	SetRMSPath("")
	ctx := &VMContext{RecordStores: map[string]*RecordStoreData{
		"image1": {Records: [][]byte{{1}}},
	}}
	ctx.saveRMS() // must not panic
	ctx.loadPersistedRMS()
}

// Server image replies must replace the same image<ID> RMS entry read by bk.o.
// Writing only a bridge-specific alias leaves the embedded placeholder active.
func TestDownloadedIconUsesCanonicalImageStore(t *testing.T) {
	ctx := &VMContext{RecordStores: map[string]*RecordStoreData{
		"image4002": {Records: [][]byte{{1, 2, 3}}},
	}}

	ctx.storeDownloadedIcon(4002, []byte{9, 8, 7, 6})

	store, ok := ctx.RecordStores["image4002"]
	if !ok || store == nil || len(store.Records) != 1 || !bytes.Equal(store.Records[0], []byte{0xF6, 0xF7, 0xF8, 0xF9}) {
		t.Fatalf("canonical image4002 store = %#v, want XOR-encoded downloaded bytes", store)
	}
	if got := ctx.loadDownloadedIcon(4002); !bytes.Equal(got, []byte{9, 8, 7, 6}) {
		t.Fatalf("loadDownloadedIcon(4002) = %v, want downloaded bytes", got)
	}
}
