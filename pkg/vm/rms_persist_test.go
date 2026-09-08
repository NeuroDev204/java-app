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
	// The embedded baseline must still be there alongside the restored stores.
	if _, ok := second.RecordStores["image4000"]; !ok {
		t.Fatal("embedded baseline store image4000 was lost")
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
