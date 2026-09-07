package upgradebot

type Config struct {
	TargetLevel       int8  `json:"target_level"`
	Clover3Min        int8  `json:"clover3_min"`
	Clover3Max        int8  `json:"clover3_max"`
	Clover4Min        int8  `json:"clover4_min"`
	Clover4Max        int8  `json:"clover4_max"`
	StopIfOutOfClover bool  `json:"stop_if_out_of_clover"`
	DelayMs           int   `json:"delay_ms"`
	ItemID            int16 `json:"item_id"`
	ItemType          int8  `json:"item_type"`
}

type Decision struct {
	CanProceed bool
	UseClover3 bool
	UseClover4 bool
	Reason     string
}

func Evaluate(cfg Config, level int8, snap Snapshot) Decision {
	if level >= cfg.TargetLevel {
		return Decision{Reason: "Target level reached"}
	}
	need3 := level >= cfg.Clover3Min && level <= cfg.Clover3Max
	need4 := level >= cfg.Clover4Min && level <= cfg.Clover4Max
	if need3 && snap.Clover3() == nil {
		if cfg.StopIfOutOfClover {
			return Decision{Reason: "Out of Clover 3"}
		}
		return Decision{CanProceed: true, Reason: "Ready to upgrade"}
	}
	if need4 && snap.Clover4() == nil {
		if cfg.StopIfOutOfClover {
			return Decision{Reason: "Out of Clover 4"}
		}
		return Decision{CanProceed: true, Reason: "Ready to upgrade"}
	}
	return Decision{
		CanProceed: true,
		UseClover3: need3 && snap.Clover3() != nil,
		UseClover4: need4 && snap.Clover4() != nil,
		Reason:     "Ready to upgrade",
	}
}
