package vm

import (
	"testing"
)

func TestEmbeddedRMSPreloaded(t *testing.T) {
	stores := getBaselineRMSStores()
	if stores == nil {
		t.Fatal("expected RecordStores to be non-nil")
	}

	requiredStores := []string{"image3005", "image3023", "image3029", "image3044"}
	for _, name := range requiredStores {
		store, exists := stores[name]
		if !exists {
			t.Fatalf("expected preloaded store %s to exist", name)
		}
		if len(store.Records) == 0 {
			t.Fatalf("expected store %s to contain records, got 0", name)
		}
		if len(store.Records[0]) == 0 {
			t.Fatalf("expected store %s record 0 to contain bytes, got 0", name)
		}
	}
}
