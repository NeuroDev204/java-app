package upgradebot

import "testing"

func TestEvaluateCloverRanges(t *testing.T) {
	cfg := Config{TargetLevel: 10, Clover3Min: 4, Clover3Max: 7, Clover4Min: 8, Clover4Max: 15, StopIfOutOfClover: true}
	snap := Snapshot{Items: []*Item{
		{ID: 10, Name: "Co 3 la", Quantity: 2, IsClover3: true},
		{ID: 11, Name: "Co 4 la", Quantity: 1, IsClover4: true},
	}}
	d := Evaluate(cfg, 6, snap)
	if !d.CanProceed || !d.UseClover3 || d.UseClover4 {
		t.Fatalf("%+v", d)
	}
	d = Evaluate(cfg, 8, snap)
	if !d.CanProceed || !d.UseClover4 {
		t.Fatalf("%+v", d)
	}
	d = Evaluate(cfg, 10, snap)
	if d.CanProceed {
		t.Fatal("target reached")
	}
}

func TestEvaluateStopsWhenOutOfClover(t *testing.T) {
	cfg := Config{TargetLevel: 10, Clover3Min: 4, Clover3Max: 7, StopIfOutOfClover: true}
	d := Evaluate(cfg, 5, Snapshot{})
	if d.CanProceed {
		t.Fatal("should stop")
	}
	cfg.StopIfOutOfClover = false
	d = Evaluate(cfg, 5, Snapshot{})
	if !d.CanProceed || d.UseClover3 {
		t.Fatalf("%+v", d)
	}
}
