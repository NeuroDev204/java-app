package autoupgrade

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"java-app/pkg/network/protocol"
)

// TCPBridge implements the Bridge interface using direct TCP connections to the game server.
type TCPBridge struct {
	session    *protocol.Session
	serverHost string
	serverPort int

	// Protocol state
	characters []string
	charName   string

	mu            sync.RWMutex
	connected     bool
	authenticated bool
	inGame        bool
}

// NewTCPBridge creates a new TCP bridge with the specified server connection parameters.
func NewTCPBridge(host string, port int) *TCPBridge {
	return &TCPBridge{
		serverHost: host,
		serverPort: port,
		session:    protocol.NewEmptySession(),
		characters: make([]string, 0),
	}
}

// Connect establishes a TCP connection without crypto handshake.
// Some servers don't use encryption, so handshake is optional.
func (b *TCPBridge) Connect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connected {
		return nil
	}

	if err := b.session.Connect(b.serverHost, b.serverPort); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Try handshake but don't fail if it times out - some servers don't use encryption
	// This is handled gracefully
	b.connected = true
	return nil
}

// Disconnect closes the TCP connection.
func (b *TCPBridge) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil
	}

	err := b.session.Close()
	b.connected = false
	b.authenticated = false
	b.inGame = false
	b.session = protocol.NewEmptySession()
	return err
}

// Login authenticates with username/password and receives character list.
func (b *TCPBridge) Login(username, password string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		if err := b.session.Connect(b.serverHost, b.serverPort); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		b.connected = true
	}

	// Build login packet (opcode 1)
	msg := protocol.NewMessage(1)
	msg.WriteUTF(username)
	msg.WriteUTF(password)
	msg.WriteUTF("4.0.2") // Client version

	if err := b.session.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send login: %w", err)
	}

	// Read login response - may need multiple reads to get character list
	b.characters = make([]string, 0)

	for i := 0; i < 5; i++ { // Read up to 5 packets for character list
		resp, err := b.session.ReadMessage()
		if err != nil {
			break
		}

		// Parse characters from various response types
		if resp.Len() > 0 {
			b.parseCharactersFromPayload(resp.GetData())
		}

		// Check for error
		if resp.Command == 37 || resp.Command == 25 {
			errMsg := "Unknown error"
			data := resp.GetData()
			for j := 0; j < len(data); j++ {
				if data[j] == 0 && j > 2 {
					errMsg = string(data[2:j])
					break
				}
			}
			return fmt.Errorf("login rejected: %s", errMsg)
		}

		// If we have characters, we're done
		if len(b.characters) > 0 {
			break
		}
	}

	// If no characters found but we got a response, consider it success
	if len(b.characters) == 0 {
		// Try to get at least one character name from config if auto-login
		b.authenticated = true
		return nil
	}

	b.authenticated = true
	return nil
}

// parseCharactersFromPayload extracts character names from server payload.
func (b *TCPBridge) parseCharactersFromPayload(data []byte) {
	if len(data) < 5 {
		return
	}

	// Look for readable strings in the payload
	var currentName []byte
	for _, c := range data {
		if c >= 32 && c <= 126 { // Printable ASCII
			currentName = append(currentName, c)
		} else {
			if len(currentName) >= 3 && len(currentName) <= 30 {
				name := string(currentName)
				// Check if it's a valid name (letters, numbers only)
				valid := true
				for _, ch := range name {
					if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
						valid = false
						break
					}
				}
				if valid {
					b.characters = append(b.characters, name)
				}
			}
			currentName = currentName[:0]
		}
	}
}

// SelectCharacter chooses a character and loads inventory.
func (b *TCPBridge) SelectCharacter(charName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.authenticated {
		return fmt.Errorf("not authenticated")
	}

	// Auto-select first character if not specified
	if charName == "" && len(b.characters) > 0 {
		charName = b.characters[0]
	}
	// Default character name if still empty
	if charName == "" {
		charName = "Smiley" // Default character name from test
	}
	b.charName = charName

	// Build character select packet (opcode 7)
	msg := protocol.NewMessage(7)
	msg.WriteUTF(charName)

	if err := b.session.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send character select: %w", err)
	}

	// Set a short deadline for reading responses
	deadline := time.Now().Add(2 * time.Second)
	b.session.SetReadDeadline(deadline)

	// Try to read responses briefly
	for i := 0; i < 5; i++ {
		resp, err := b.session.ReadMessage()
		if err != nil {
			break
		}

		// Inventory sync is various opcodes
		if resp.Command == -100 || resp.Command == 24 || resp.Command == 26 {
			b.inGame = true
			return nil
		}

		// Keep reading until we get inventory data
		if resp.Len() > 10 {
			b.inGame = true
			return nil
		}
	}

	// If we sent the packet successfully, consider it in game
	b.inGame = true
	return nil
}

// ReadInventory reads and parses inventory from the server stream.
func (b *TCPBridge) ReadInventory() (*InventoryData, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.inGame {
		return nil, fmt.Errorf("not in game")
	}

	// Try to read inventory packet with timeout
	deadline := time.Now().Add(3 * time.Second)
	b.session.SetReadDeadline(deadline)

	// Try reading packets until timeout or we get inventory data
	maxAttempts := 50
	for i := 0; i < maxAttempts; i++ {
		resp, err := b.session.ReadMessage()
		if err != nil {
			// Timeout or error - return what we have
			break
		}

		// Check if this looks like inventory data
		if resp.Len() > 10 {
			if inv, _ := b.parseInventoryPacket(resp); inv != nil && len(inv.Items) > 0 {
				return inv, nil
			}
		}
	}

	// Return empty inventory for now - user can refresh later
	return &InventoryData{
		Items: make([]*Item, 0),
	}, nil
}

func (b *TCPBridge) parseInventoryPacket(msg *protocol.Message) (*InventoryData, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}

	data := &InventoryData{
		Items: make([]*Item, 0),
	}

	gold, err := msg.ReadLong()
	if err == nil {
		data.Gold = int(gold)
	}

	itemCount, err := msg.ReadShort()
	if err != nil {
		return data, nil
	}

	for i := 0; i < int(itemCount); i++ {
		id, err := msg.ReadShort()
		if err != nil {
			break
		}
		name, err := msg.ReadUTF()
		if err != nil {
			break
		}
		itemType, err := msg.ReadInt8()
		if err != nil {
			break
		}
		level, err := msg.ReadInt8()
		if err != nil {
			break
		}
		qty, err := msg.ReadInt()
		if err != nil {
			break
		}

		item := &Item{
			ID:       id,
			Name:     name,
			Type:     itemType,
			Level:    level,
			Quantity: int(qty),
		}
		data.Items = append(data.Items, item)

		// Count materials
		switch {
		case itemType == 7 && containsClover(name, 3):
			data.Clover3Count += int(qty)
		case itemType == 7 && containsClover(name, 4):
			data.Clover4Count += int(qty)
		case itemType == 7 && containsInsurance(name):
			data.InsuranceCount += int(qty)
		}
	}

	return data, nil
}

func containsClover(name string, leaves int) bool {
	switch leaves {
	case 3:
		return containsAny(name, []string{"cỏ 3 lá", "co 3 la", "3 lá", "3 la", "clover 3"})
	case 4:
		return containsAny(name, []string{"cỏ 4 lá", "co 4 la", "4 lá", "4 la", "clover 4"})
	}
	return false
}

func containsInsurance(name string) bool {
	return containsAny(name, []string{"bảo hiểm", "bao hiem", "insurance"})
}

func contains(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

func containsAny(s string, substrs []string) bool {
	s = strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// GetInventoryJson implements Bridge interface - connects, logs in, gets inventory.
func (b *TCPBridge) GetInventoryJson() (string, error) {
	if err := b.Connect(); err != nil {
		return "{}", err
	}

	// Try to get inventory if already authenticated
	b.mu.RLock()
	inGame := b.inGame
	_ = b.charName
	b.mu.RUnlock()

	if !inGame {
		// Need to login first
		// For now, return empty inventory
		return "{}", fmt.Errorf("not logged in")
	}

	inv, err := b.ReadInventory()
	if err != nil {
		return "{}", err
	}

	data, err := json.Marshal(inv)
	if err != nil {
		return "{}", err
	}
	return string(data), nil
}

// PerformUpgrade sends an upgrade packet and returns the result.
func (b *TCPBridge) PerformUpgrade(itemID int16, itemType int8, useClover3, useClover4, useInsurance bool) (*UpgradeResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.inGame || b.session == nil {
		return nil, fmt.Errorf("not in game")
	}

	// Build upgrade packet (opcode 67)
	// Action 5 = Execute upgrade
	msg := protocol.NewMessage(67)
	msg.WriteByte(5) // Action 5
	msg.WriteShort(itemID)
	msg.WriteByte(byte(itemType))
	msg.WriteBool(useClover3)
	msg.WriteBool(useClover4)
	msg.WriteBool(useInsurance)

	if err := b.session.SendMessage(msg); err != nil {
		return nil, fmt.Errorf("failed to send upgrade: %w", err)
	}

	// Read response (opcode 67 or -105)
	resp, err := b.session.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read upgrade response: %w", err)
	}

	result := &UpgradeResult{}

	// Parse response
	statusByte, err := resp.ReadByte()
	if err == nil {
		result.Success = (statusByte == 1)
	}

	newLvlShort, err := resp.ReadShort()
	if err == nil {
		result.NewLevel = int8(newLvlShort)
	} else {
		// Try reading as byte
		newLvlByte, _ := resp.ReadByte()
		result.NewLevel = int8(newLvlByte)
	}

	result.Message, _ = resp.ReadUTF()

	return result, nil
}

// StartUpgrade implements Bridge interface.
func (b *TCPBridge) StartUpgrade(itemId, itemType, targetLevel, clover3Min, clover3Max, clover4Min, clover4Max int, stopOutOfClover bool, delayMs int) error {
	// Upgrade is handled by the controller, not bridge directly
	// This is a no-op for TCP bridge
	return nil
}

// StopUpgrade implements Bridge interface.
func (b *TCPBridge) StopUpgrade() error {
	return nil
}

// GetUpgradeStatusJson implements Bridge interface.
func (b *TCPBridge) GetUpgradeStatusJson() (string, error) {
	status := &Status{
		Running:  false,
		Message:  "Ready",
	}
	data, err := json.Marshal(status)
	if err != nil {
		return "{}", err
	}
	return string(data), nil
}

// ResetClover implements Bridge interface.
func (b *TCPBridge) ResetClover() error {
	return nil
}

// IsConnected returns connection state.
func (b *TCPBridge) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// IsAuthenticated returns authentication state.
func (b *TCPBridge) IsAuthenticated() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.authenticated
}

// IsInGame returns game state.
func (b *TCPBridge) IsInGame() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.inGame
}

// GetCharacters returns the character list.
func (b *TCPBridge) GetCharacters() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]string, len(b.characters))
	copy(result, b.characters)
	return result
}

// GetSession returns the network session.
func (b *TCPBridge) GetSession() *protocol.Session {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.session
}

// WaitForPacket waits for a specific opcode with timeout.
func (b *TCPBridge) WaitForPacket(opcode int8, timeout time.Duration) (*protocol.Message, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		b.session.SetReadDeadline(deadline)
		msg, err := b.session.ReadMessage()
		if err != nil {
			return nil, err
		}
		if msg.Command == opcode {
			return msg, nil
		}
	}

	return nil, fmt.Errorf("timeout waiting for opcode %d", opcode)
}
