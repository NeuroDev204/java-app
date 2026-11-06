//go:build !cgo

package autoupgrade

type NoopBridge struct{}

func (b *NoopBridge) GetInventoryJson() (string, error) {
	return `{"gold":0,"clover3Count":0,"clover4Count":0,"insuranceCount":0,"items":[]}`, nil
}

func (b *NoopBridge) StartUpgrade(itemId, itemType, targetLevel, clover3Min, clover3Max, clover4Min, clover4Max int, stopOutOfClover bool, delayMs int) error {
	return nil
}

func (b *NoopBridge) StopUpgrade() error {
	return nil
}

func (b *NoopBridge) GetUpgradeStatusJson() (string, error) {
	return `{"running":false,"connected":false,"in_game":false,"currentLevel":0,"targetLevel":0,"attempts":0,"successes":0,"failures":0,"message":"","stopped":false}`, nil
}

func (b *NoopBridge) ResetClover() error {
	return nil
}

func defaultBridge() Bridge {
	return &NoopBridge{}
}
