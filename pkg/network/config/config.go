// Package config provides configuration models, default settings, and serialization
// utilities for the knight-auto-upgrade bot.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerConfig defines the TCP connection target for the game server.
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// AccountConfig defines player authentication credentials and character name.
type AccountConfig struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Character string `json:"character"`
}

// UpgradeConfig defines the automated equipment upgrade rules, clover and insurance brackets,
// and safety guards.
type UpgradeConfig struct {
	TargetLevel       int8   `json:"target_level"`
	Clover3Levels     []int  `json:"clover_3_levels"`
	Clover4Levels     []int  `json:"clover_4_levels"`
	InsuranceLevels   []int  `json:"insurance_levels"`
	StopIfOutOfClover bool   `json:"stop_if_out_of_clover"`
	DelayMs           int    `json:"delay_ms"`
	ItemName          string `json:"item_name,omitempty"`
	ItemID            int16  `json:"item_id,omitempty"`
	ItemType          int8   `json:"item_type,omitempty"`
}

// RequiresClover3 returns true if the specified level is configured to use Cỏ 3 lá.
func (u *UpgradeConfig) RequiresClover3(level int8) bool {
	for _, l := range u.Clover3Levels {
		if int8(l) == level {
			return true
		}
	}
	return false
}

// RequiresClover4 returns true if the specified level is configured to use Cỏ 4 lá.
func (u *UpgradeConfig) RequiresClover4(level int8) bool {
	for _, l := range u.Clover4Levels {
		if int8(l) == level {
			return true
		}
	}
	return false
}

// RequiresInsurance returns true if the specified level is configured to use Bảo hiểm.
func (u *UpgradeConfig) RequiresInsurance(level int8) bool {
	for _, l := range u.InsuranceLevels {
		if int8(l) == level {
			return true
		}
	}
	return false
}

// Config wraps all system configuration sections.
type Config struct {
	Server  ServerConfig  `json:"server"`
	Account AccountConfig `json:"account"`
	Upgrade UpgradeConfig `json:"upgrade"`
}

// DefaultConfig returns a new Config populated with recommended standard settings.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "hs1.teamobi.com",
			Port: 19129,
		},
		Account: AccountConfig{
			Username:  "",
			Password:  "",
			Character: "",
		},
		Upgrade: UpgradeConfig{
			TargetLevel:       10,
			Clover3Levels:     []int{4, 5, 6, 7},
			Clover4Levels:     []int{8, 9, 10},
			InsuranceLevels:   []int{8, 9, 10},
			StopIfOutOfClover: true,
			DelayMs:           250,
			ItemName:          "",
			ItemID:            0,
			ItemType:          0,
		},
	}
}

// LoadConfig reads and parses the JSON configuration file from the specified path.
// It returns an error if reading or JSON parsing fails.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config at %s: %w", path, err)
	}

	return cfg, nil
}

// SaveConfig serializes the given configuration to JSON with indentation and writes it to disk.
func SaveConfig(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil config")
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", path, err)
	}

	return nil
}
