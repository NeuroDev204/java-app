package autoupgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TCPConfig represents the configuration for TCP auto-upgrade mode.
type TCPConfig struct {
	Server  TCPConfigServer  `json:"server"`
	Account TCPConfigAccount `json:"account"`
	Upgrade TCPConfigUpgrade `json:"upgrade"`
}

// TCPConfigServer holds server connection settings.
type TCPConfigServer struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// TCPConfigAccount holds account credentials.
type TCPConfigAccount struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Character string `json:"character"`
}

// TCPConfigUpgrade holds upgrade rules.
type TCPConfigUpgrade struct {
	TargetLevel        int8  `json:"target_level"`
	Clover3Min       int8  `json:"clover_3_min"`
	Clover3Max       int8  `json:"clover_3_max"`
	Clover4Min       int8  `json:"clover_4_min"`
	Clover4Max       int8  `json:"clover_4_max"`
	StopIfOutOfClover bool  `json:"stop_if_out_of_clover"`
	DelayMs          int   `json:"delay_ms"`
}

// LoadTCPConfig loads TCP configuration from a JSON file.
func LoadTCPConfig(path string) (*TCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg TCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// DefaultTCPConfig returns default TCP configuration.
func DefaultTCPConfig() *TCPConfig {
	return &TCPConfig{
		Server: TCPConfigServer{
			Host: "hs1.teamobi.com",
			Port: 19129,
		},
		Account: TCPConfigAccount{
			Username:  "",
			Password: "",
			Character: "",
		},
		Upgrade: TCPConfigUpgrade{
			TargetLevel:        10,
			Clover3Min:       4,
			Clover3Max:       7,
			Clover4Min:       8,
			Clover4Max:       15,
			StopIfOutOfClover: true,
			DelayMs:          500,
		},
	}
}

// UpgradeDecision contains the outcome of evaluating the current gear level against configured rules.
type UpgradeDecision struct {
	UseClover3   bool   `json:"use_clover_3"`
	UseClover4   bool   `json:"use_clover_4"`
	UseInsurance bool   `json:"use_insurance"`
	CanProceed   bool   `json:"can_proceed"`
	Reason       string `json:"reason"`
}

// UpgradeRuleEngine evaluates equipment levels against configurable upgrade rules.
type UpgradeRuleEngine struct {
	mu  sync.RWMutex
	cfg *UpgradeConfig
}

// NewUpgradeRuleEngine constructs a new rule engine.
func NewUpgradeRuleEngine(cfg *UpgradeConfig) *UpgradeRuleEngine {
	return &UpgradeRuleEngine{cfg: cfg}
}

// EvaluateForLevel evaluates the upgrade attempt at the given level.
func (r *UpgradeRuleEngine) EvaluateForLevel(currentLevel int8, inv *InventoryData) (UpgradeDecision, error) {
	if r == nil {
		return UpgradeDecision{}, fmt.Errorf("upgrade rule engine is nil")
	}
	r.mu.RLock()
	cfg := r.cfg
	r.mu.RUnlock()

	if cfg == nil {
		return UpgradeDecision{}, fmt.Errorf("upgrade config is nil")
	}

	if currentLevel >= cfg.TargetLevel {
		return UpgradeDecision{CanProceed: false, Reason: "Target level reached"}, nil
	}

	decision := UpgradeDecision{CanProceed: true, Reason: "Ready to upgrade"}

	requiresC3 := currentLevel >= cfg.Clover3Min && currentLevel <= cfg.Clover3Max
	requiresC4 := currentLevel >= cfg.Clover4Min && currentLevel <= cfg.Clover4Max

	if requiresC3 && inv.Clover3Count > 0 {
		decision.UseClover3 = true
	} else if requiresC3 && cfg.StopIfOutOfClover {
		return UpgradeDecision{CanProceed: false, Reason: "Out of Clover 3"}, nil
	}

	if requiresC4 && inv.Clover4Count > 0 {
		decision.UseClover4 = true
	} else if requiresC4 && cfg.StopIfOutOfClover {
		return UpgradeDecision{CanProceed: false, Reason: "Out of Clover 4"}, nil
	}

	if currentLevel >= cfg.InsuranceMin && inv.InsuranceCount > 0 {
		decision.UseInsurance = true
	}

	return decision, nil
}

// UpgradeConfig defines the automated equipment upgrade rules.
type UpgradeConfig struct {
	TargetLevel       int8  `json:"target_level"`
	Clover3Min       int8  `json:"clover_3_min"`
	Clover3Max       int8  `json:"clover_3_max"`
	Clover4Min       int8  `json:"clover_4_min"`
	Clover4Max       int8  `json:"clover_4_max"`
	InsuranceMin      int8  `json:"insurance_min"`
	StopIfOutOfClover bool  `json:"stop_if_out_of_clover"`
	DelayMs          int   `json:"delay_ms"`
}

// UpgradeStats tracks metrics for an upgrade session.
type UpgradeStats struct {
	Attempts      int   `json:"attempts"`
	Successes     int   `json:"successes"`
	Failures      int   `json:"failures"`
	StartLevel    int8  `json:"start_level"`
	CurrentLevel  int8  `json:"current_level"`
	TargetLevel   int8  `json:"target_level"`
	Clover3Used   int   `json:"clover_3_used"`
	Clover4Used   int   `json:"clover_4_used"`
	InsuranceUsed int   `json:"insurance_used"`
}

// SuccessRate returns the percentage of successful upgrade attempts.
func (s UpgradeStats) SuccessRate() float64 {
	if s.Attempts <= 0 {
		return 0.0
	}
	return (float64(s.Successes) / float64(s.Attempts)) * 100.0
}

// UpgradeResult captures the outcome of a single upgrade attempt.
type UpgradeResult struct {
	Success       bool   `json:"success"`
	OldLevel      int8   `json:"old_level"`
	NewLevel      int8   `json:"new_level"`
	ItemName      string `json:"item_name"`
	ItemID        int16  `json:"item_id"`
	UsedClover3   bool   `json:"used_clover_3"`
	UsedClover4   bool   `json:"used_clover_4"`
	UsedInsurance bool   `json:"used_insurance"`
	Message       string `json:"message"`
}

// TCPController coordinates the auto-upgrade logic with TCP bridge.
type TCPController struct {
	bridge        *TCPBridge
	mu            sync.RWMutex
	settings      *UpgradeSettings
	lastInventory *InventoryData
	lastStatus    *Status
	rules         *UpgradeRuleEngine
	stats         UpgradeStats
	statsMu       sync.RWMutex
	upgradeCtx    context.Context
	upgradeCancel context.CancelFunc
	upgrading     bool
	upgradeMu     sync.Mutex
}

// NewTCPController creates a new TCP controller with the given bridge.
func NewTCPController(bridge *TCPBridge) *TCPController {
	cfg := &UpgradeConfig{
		TargetLevel:       8,
		Clover3Min:       4,
		Clover3Max:       7,
		Clover4Min:       8,
		Clover4Max:       15,
		InsuranceMin:      8,
		StopIfOutOfClover: true,
		DelayMs:          500,
	}
	return &TCPController{
		bridge:   bridge,
		settings: DefaultSettings(),
		rules:    NewUpgradeRuleEngine(cfg),
	}
}

// RefreshInventory fetches inventory from the server.
func (c *TCPController) RefreshInventory() (interface{}, error) {
	if c.bridge == nil {
		return nil, fmt.Errorf("bridge is nil")
	}

	if !c.bridge.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	inv, err := c.bridge.ReadInventory()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.lastInventory = inv
	c.mu.Unlock()

	return inv, nil
}

// GetCachedInventory returns cached inventory.
func (c *TCPController) GetCachedInventory() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastInventory == nil {
		return nil
	}
	invCopy := *c.lastInventory
	return &invCopy
}

// GetSettings returns a copy of current settings.
func (c *TCPController) GetSettings() *UpgradeSettings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.settings == nil {
		return DefaultSettings()
	}
	sCopy := *c.settings
	return &sCopy
}

// UpdateSettings updates configuration settings.
func (c *TCPController) UpdateSettings(s *UpgradeSettings) {
	if s == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	sCopy := *s
	c.settings = &sCopy

	// Update rule engine config
	cfg := &UpgradeConfig{
		TargetLevel:        s.TargetLevel,
		Clover3Min:        s.Clover3Min,
		Clover3Max:        s.Clover3Max,
		Clover4Min:        s.Clover4Min,
		Clover4Max:        s.Clover4Max,
		InsuranceMin:       8, // Default
		StopIfOutOfClover: s.StopIfOutOfClover,
		DelayMs:           s.DelayMs,
	}
	c.rules.UpdateConfig(cfg)
}

// IsRunning indicates if the auto-upgrade routine is currently active.
func (c *TCPController) IsRunning() bool {
	c.upgradeMu.Lock()
	defer c.upgradeMu.Unlock()
	return c.upgrading
}

// StartUpgrade initiates the auto-upgrade routine.
func (c *TCPController) StartUpgrade(settings *UpgradeSettings) error {
	if settings == nil {
		settings = c.GetSettings()
	}

	c.upgradeMu.Lock()
	if c.upgrading {
		c.upgradeMu.Unlock()
		return fmt.Errorf("upgrade already running")
	}
	c.upgrading = true
	c.upgradeMu.Unlock()

	c.upgradeCtx, c.upgradeCancel = context.WithCancel(context.Background())
	c.UpdateSettings(settings)

	// Reset stats
	c.statsMu.Lock()
	c.stats = UpgradeStats{TargetLevel: settings.TargetLevel}
	c.statsMu.Unlock()

	// Start upgrade in background
	go c.runUpgradeLoop()

	return nil
}

// StopUpgrade requests the auto-upgrade routine to halt.
func (c *TCPController) StopUpgrade() error {
	c.upgradeMu.Lock()
	defer c.upgradeMu.Unlock()

	if !c.upgrading {
		return nil
	}

	if c.upgradeCancel != nil {
		c.upgradeCancel()
	}
	c.upgrading = false

	// Update status
	c.mu.Lock()
	if c.lastStatus == nil {
		c.lastStatus = &Status{}
	}
	c.lastStatus.Running = false
	c.lastStatus.Stopped = true
	c.mu.Unlock()

	return nil
}

// GetStatus returns the current upgrade status.
func (c *TCPController) GetStatus() (*Status, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastStatus, nil
}

// GetCachedStatus returns the cached status.
func (c *TCPController) GetCachedStatus() *Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus == nil {
		return nil
	}
	statusCopy := *c.lastStatus
	return &statusCopy
}

// GetStats returns the current upgrade statistics.
func (c *TCPController) GetStats() UpgradeStats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats
}

func (c *TCPController) runUpgradeLoop() {
	defer func() {
		c.upgradeMu.Lock()
		c.upgrading = false
		c.upgradeMu.Unlock()
	}()

	settings := c.GetSettings()
	rawInv := c.GetCachedInventory()
	inv, ok := rawInv.(*InventoryData)
	if !ok || inv == nil {
		c.updateStatus(false, "No inventory data")
		return
	}

	// Find target item
	var targetItem *Item
	if settings.SelectedItemIndex >= 0 && settings.SelectedItemIndex < len(inv.Items) {
		targetItem = inv.Items[settings.SelectedItemIndex]
	}

	if targetItem == nil {
		c.updateStatus(false, "No target item selected")
		return
	}

	c.statsMu.Lock()
	c.stats.StartLevel = targetItem.Level
	c.stats.CurrentLevel = targetItem.Level
	c.statsMu.Unlock()

	delay := time.Duration(settings.DelayMs) * time.Millisecond
	if delay < 100*time.Millisecond {
		delay = 500 * time.Millisecond
	}

	for {
		select {
		case <-c.upgradeCtx.Done():
			c.updateStatus(false, "Stopped by user")
			return
		default:
		}

		currentLevel := targetItem.Level
		if currentLevel >= settings.TargetLevel {
			c.updateStatus(false, fmt.Sprintf("Target +%d reached!", settings.TargetLevel))
			return
		}

		// Refresh inventory
		rawInv, err := c.RefreshInventory()
		if err != nil {
			c.updateStatus(false, fmt.Sprintf("Inventory error: %v", err))
			return
		}
		inv, ok := rawInv.(*InventoryData)
		if !ok || inv == nil {
			c.updateStatus(false, "Invalid inventory data")
			return
		}

		// Evaluate decision
		dec, err := c.rules.EvaluateForLevel(currentLevel, inv)
		if err != nil {
			c.updateStatus(false, fmt.Sprintf("Rule error: %v", err))
			return
		}

		if !dec.CanProceed {
			c.updateStatus(false, dec.Reason)
			return
		}

		// Send upgrade packet
		result, err := c.bridge.PerformUpgrade(targetItem.ID, targetItem.Type, dec.UseClover3, dec.UseClover4, dec.UseInsurance)
		if err != nil {
			c.updateStatus(false, fmt.Sprintf("Upgrade error: %v", err))
			return
		}

		// Update stats
		c.statsMu.Lock()
		c.stats.Attempts++
		if result.Success {
			c.stats.Successes++
		} else {
			c.stats.Failures++
		}
		if dec.UseClover3 {
			c.stats.Clover3Used++
		}
		if dec.UseClover4 {
			c.stats.Clover4Used++
		}
		if dec.UseInsurance {
			c.stats.InsuranceUsed++
		}
		c.stats.CurrentLevel = result.NewLevel
		c.statsMu.Unlock()

		// Update item level
		targetItem.Level = result.NewLevel

		// Update status
		msg := result.Message
		if msg == "" && result.Success {
			msg = "Success!"
		}
		c.updateStatus(true, msg)

		select {
		case <-c.upgradeCtx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func (c *TCPController) updateStatus(running bool, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lastStatus == nil {
		c.lastStatus = &Status{}
	}

	c.statsMu.RLock()
	stats := c.stats
	c.statsMu.RUnlock()

	c.lastStatus.Running = running
	c.lastStatus.Message = message
	c.lastStatus.Attempts = stats.Attempts
	c.lastStatus.Successes = stats.Successes
	c.lastStatus.Failures = stats.Failures
	c.lastStatus.CurrentLevel = stats.CurrentLevel
}

// UpdateConfig updates the rule engine configuration.
func (r *UpgradeRuleEngine) UpdateConfig(cfg *UpgradeConfig) {
	if r == nil || cfg == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = cfg
}
