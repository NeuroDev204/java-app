package vm

import (
	_ "embed"
	"encoding/json"
	"log"
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
}

