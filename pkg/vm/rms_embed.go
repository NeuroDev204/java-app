package vm

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"sync"
)

//go:embed default_rms.json
var defaultRMSData []byte

type jsonRMSRecord struct {
	ID   int    `json:"id"`
	Data []byte `json:"data"` // encoding/json automatically base64-decodes []byte
}

type jsonRMSStore struct {
	Name    string          `json:"name"`
	Records []jsonRMSRecord `json:"records"`
}

var (
	baselineRMSOnce   sync.Once
	baselineRMSStores map[string]*RecordStoreData
)

func getBaselineRMSStores() map[string]*RecordStoreData {
	baselineRMSOnce.Do(func() {
		baselineRMSStores = make(map[string]*RecordStoreData)
		if len(defaultRMSData) == 0 {
			return
		}
		var stores []jsonRMSStore
		if err := json.Unmarshal(defaultRMSData, &stores); err != nil {
			log.Printf("[RMS-PRELOAD] failed to unmarshal embedded default_rms.json: %v", err)
			return
		}
		for _, s := range stores {
			rs := &RecordStoreData{Records: make([][]byte, len(s.Records))}
			for _, rec := range s.Records {
				idx := rec.ID - 1
				if idx >= 0 && idx < len(rs.Records) {
					rs.Records[idx] = rec.Data
				} else {
					rs.Records = append(rs.Records, rec.Data)
				}
			}
			baselineRMSStores[s.Name] = rs
		}
	})
	return baselineRMSStores
}

func cloneRecordStoreData(src *RecordStoreData) *RecordStoreData {
	if src == nil {
		return nil
	}
	dst := &RecordStoreData{Records: make([][]byte, len(src.Records))}
	for i, r := range src.Records {
		if r != nil {
			c := make([]byte, len(r))
			copy(c, r)
			dst.Records[i] = c
		}
	}
	return dst
}

var (
	rmsPersistPath string
	rmsPersistMu   sync.Mutex
)

// SetRMSPath points the record store at a writable file. Without it the mobile
// engine keeps record stores in memory only, so everything the game downloads
// during play — item icons above all — is lost on restart and has to be fetched
// again, leaving inventory slots showing the game's loading spinner. The desktop
// engine has always persisted (see pkg/jvm/callbacks.go saveRMS). Pass "" to
// disable persistence.
func SetRMSPath(path string) {
	rmsPersistMu.Lock()
	defer rmsPersistMu.Unlock()
	rmsPersistPath = path
}

// loadPersistedRMS overlays previously saved stores on top of the embedded
// baseline, so a store the player has accumulated wins over the shipped copy.
func (ctx *VMContext) loadPersistedRMS() {
	rmsPersistMu.Lock()
	path := rmsPersistPath
	rmsPersistMu.Unlock()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[RMS-PERSIST] read %s failed: %v", path, err)
		}
		return
	}
	var stores []jsonRMSStore
	if err := json.Unmarshal(data, &stores); err != nil {
		log.Printf("[RMS-PERSIST] parse %s failed: %v", path, err)
		return
	}
	for _, s := range stores {
		rs := &RecordStoreData{Records: make([][]byte, len(s.Records))}
		for _, rec := range s.Records {
			idx := rec.ID - 1
			if idx >= 0 && idx < len(rs.Records) {
				rs.Records[idx] = rec.Data
			} else {
				rs.Records = append(rs.Records, rec.Data)
			}
		}
		ctx.RecordStores[s.Name] = rs
	}
	log.Printf("[RMS-PERSIST] Restored %d record stores from %s", len(stores), path)
}

// saveRMS writes every record store back to disk.
// ponytail: rewrites the whole file on each record write; record writes are rare
// (a handful of downloaded icons per session), switch to a dirty-flag flush if
// that stops being true.
func (ctx *VMContext) saveRMS() {
	rmsPersistMu.Lock()
	defer rmsPersistMu.Unlock()
	if rmsPersistPath == "" {
		return
	}
	stores := make([]jsonRMSStore, 0, len(ctx.RecordStores))
	for name, rs := range ctx.RecordStores {
		if rs == nil {
			continue
		}
		recs := make([]jsonRMSRecord, 0, len(rs.Records))
		for i, data := range rs.Records {
			recs = append(recs, jsonRMSRecord{ID: i + 1, Data: data})
		}
		stores = append(stores, jsonRMSStore{Name: name, Records: recs})
	}
	data, err := json.Marshal(stores)
	if err != nil {
		log.Printf("[RMS-PERSIST] marshal failed: %v", err)
		return
	}
	tmp := rmsPersistPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		log.Printf("[RMS-PERSIST] write %s failed: %v", tmp, err)
		return
	}
	if err := os.Rename(tmp, rmsPersistPath); err != nil {
		log.Printf("[RMS-PERSIST] rename failed: %v", err)
	}
}

func (ctx *VMContext) loadDefaultRMS() {
	baseline := getBaselineRMSStores()
	if ctx.RecordStores == nil {
		ctx.RecordStores = make(map[string]*RecordStoreData)
	}

	count := 0
	for name, rs := range baseline {
		ctx.RecordStores[name] = cloneRecordStoreData(rs)
		count++
	}
	log.Printf("[RMS-PRELOAD] Preloaded %d record stores from embedded default_rms.json", count)
	ctx.loadPersistedRMS()
}

