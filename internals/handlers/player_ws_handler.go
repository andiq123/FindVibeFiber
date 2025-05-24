package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type WSMessage struct {
	Method    string      `json:"method"`
	Data      interface{} `json:"data"`
	Timestamp json.Number `json:"timestamp"`
}

// GetTimestamp returns the timestamp as int64, handling both integer and float values
func (m *WSMessage) GetTimestamp() int64 {
	if m.Timestamp == "" {
		return time.Now().UnixMilli()
	}

	// Try to parse as int64 first
	if ts, err := m.Timestamp.Int64(); err == nil {
		return ts
	}

	// If that fails, try to parse as float and convert to int64
	if ts, err := m.Timestamp.Float64(); err == nil {
		return int64(ts)
	}

	// If all parsing fails, return current time
	return time.Now().UnixMilli()
}

type Session struct {
	ID       string
	Username string
	Conn     *websocket.Conn
	DeviceID string
	LastPing time.Time
	mu       sync.RWMutex
	closed   bool
}

// IsAlive checks if the session is still alive
func (s *Session) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.closed
}

// Close marks the session as closed
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		if s.Conn != nil {
			s.Conn.Close()
		}
	}
}

// SendMessage safely sends a message to the session
func (s *Session) SendMessage(msg WSMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.Conn == nil {
		return ErrSessionClosed
	}

	// Set write deadline to prevent hanging
	s.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	defer s.Conn.SetWriteDeadline(time.Time{})

	return s.Conn.WriteJSON(msg)
}

// UpdatePing updates the last ping time
func (s *Session) UpdatePing() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastPing = time.Now()
}

var (
	ErrSessionClosed = fmt.Errorf("session is closed")
)

type SessionManager struct {
	sessions map[string]map[string]*Session // username -> sessionID -> Session
	mu       sync.RWMutex
	cleanup  chan string // sessionID to cleanup
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewSessionManager() *SessionManager {
	ctx, cancel := context.WithCancel(context.Background())
	sm := &SessionManager{
		sessions: make(map[string]map[string]*Session),
		cleanup:  make(chan string, 100),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start cleanup goroutine
	go sm.cleanupWorker()
	// Start health check goroutine
	go sm.healthChecker()

	return sm
}

func (sm *SessionManager) cleanupWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case sessionID := <-sm.cleanup:
			sm.removeSessionByID(sessionID)
		case <-ticker.C:
			sm.cleanupDeadSessions()
		}
	}
}

func (sm *SessionManager) healthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.checkSessionHealth()
		}
	}
}

func (sm *SessionManager) checkSessionHealth() {
	sm.mu.RLock()
	var toCheck []*Session
	for _, userSessions := range sm.sessions {
		for _, session := range userSessions {
			toCheck = append(toCheck, session)
		}
	}
	sm.mu.RUnlock()

	for _, session := range toCheck {
		if !session.IsAlive() {
			continue
		}

		// Send ping to check if connection is alive
		pingMsg := WSMessage{
			Method:    "Ping",
			Timestamp: json.Number(fmt.Sprintf("%d", time.Now().UnixMilli())),
		}

		if err := session.SendMessage(pingMsg); err != nil {
			log.Printf("Health check failed for user %s, device %s: %v",
				session.Username, session.DeviceID, err)
			session.Close()
			select {
			case sm.cleanup <- session.ID:
			default:
			}
		}
	}
}

func (sm *SessionManager) AddSession(username string, conn *websocket.Conn) *Session {
	sessionID := uuid.New().String()
	deviceID := uuid.New().String()

	session := &Session{
		ID:       sessionID,
		Username: username,
		Conn:     conn,
		DeviceID: deviceID,
		LastPing: time.Now(),
		closed:   false,
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.sessions[username] == nil {
		sm.sessions[username] = make(map[string]*Session)
	}
	sm.sessions[username][sessionID] = session

	log.Printf("Added session %s for user %s (device %s)", sessionID, username, deviceID)
	return session
}

func (sm *SessionManager) RemoveSession(session *Session) {
	if session == nil {
		return
	}

	session.Close()
	select {
	case sm.cleanup <- session.ID:
	default:
		// If cleanup channel is full, do it directly
		sm.removeSessionByID(session.ID)
	}
}

func (sm *SessionManager) removeSessionByID(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for username, userSessions := range sm.sessions {
		if session, exists := userSessions[sessionID]; exists {
			log.Printf("Removing session %s for user %s (device %s)",
				sessionID, username, session.DeviceID)
			delete(userSessions, sessionID)

			if len(userSessions) == 0 {
				delete(sm.sessions, username)
				log.Printf("No more sessions for user %s", username)
			}
			break
		}
	}
}

func (sm *SessionManager) cleanupDeadSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for username, userSessions := range sm.sessions {
		for sessionID, session := range userSessions {
			if !session.IsAlive() {
				log.Printf("Cleaning up dead session %s for user %s", sessionID, username)
				delete(userSessions, sessionID)
			}
		}

		if len(userSessions) == 0 {
			delete(sm.sessions, username)
		}
	}
}

func (sm *SessionManager) GetActiveSessions(username string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []*Session
	if userSessions, exists := sm.sessions[username]; exists {
		for _, session := range userSessions {
			if session.IsAlive() {
				sessions = append(sessions, session)
			}
		}
	}
	return sessions
}

func (sm *SessionManager) GetAllUsers() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	users := make([]string, 0, len(sm.sessions))
	for username := range sm.sessions {
		users = append(users, username)
	}
	return users
}

func (sm *SessionManager) BroadcastToUserExcept(username string, msg WSMessage, excludeSession *Session) {
	sessions := sm.GetActiveSessions(username)

	for _, session := range sessions {
		if session == excludeSession {
			continue
		}

		if err := session.SendMessage(msg); err != nil {
			log.Printf("Failed to send message to user %s, device %s: %v",
				session.Username, session.DeviceID, err)
			session.Close()
			select {
			case sm.cleanup <- session.ID:
			default:
			}
		}
	}
}

func (sm *SessionManager) BroadcastExcept(msg WSMessage, excludeSession *Session) {
	sm.mu.RLock()
	var allSessions []*Session
	for _, userSessions := range sm.sessions {
		for _, session := range userSessions {
			if session != excludeSession && session.IsAlive() {
				allSessions = append(allSessions, session)
			}
		}
	}
	sm.mu.RUnlock()

	for _, session := range allSessions {
		if err := session.SendMessage(msg); err != nil {
			log.Printf("Failed to broadcast to user %s, device %s: %v",
				session.Username, session.DeviceID, err)
			session.Close()
			select {
			case sm.cleanup <- session.ID:
			default:
			}
		}
	}
}

func (sm *SessionManager) Shutdown() {
	sm.cancel()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, userSessions := range sm.sessions {
		for _, session := range userSessions {
			session.Close()
		}
	}

	close(sm.cleanup)
}

// Global session manager
var sessionManager = NewSessionManager()

func PlayerWebSocketHandler() func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		var currentSession *Session

		// Set connection settings
		c.SetReadDeadline(time.Time{}) // No read deadline
		c.SetPongHandler(func(string) error {
			if currentSession != nil {
				currentSession.UpdatePing()
			}
			return nil
		})

		defer func() {
			if currentSession != nil {
				sessionManager.RemoveSession(currentSession)
				// Broadcast updated user list
				broadcastSessionsList()
			}
		}()

		for {
			_, msgBytes, err := c.ReadMessage()
			if err != nil {
				if currentSession != nil {
					log.Printf("Read error for user %s: %v", currentSession.Username, err)
				}
				break
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
				if currentSession != nil {
					log.Printf("Unmarshal error for user %s: %v", currentSession.Username, err)
				}
				continue
			}

			if err := handleMessage(c, &wsMsg, &currentSession); err != nil {
				log.Printf("Message handling error: %v", err)
				break
			}
		}
	}
}

// Add this to your existing handler.go file

func handleMessage(c *websocket.Conn, msg *WSMessage, currentSession **Session) error {
	switch msg.Method {
	case "Connect":
		username, ok := msg.Data.(string)
		if !ok || username == "" {
			return fmt.Errorf("invalid username in connect message")
		}

		*currentSession = sessionManager.AddSession(username, c)
		log.Printf("User %s connected from device %s", username, (*currentSession).DeviceID)

		confirmMsg := WSMessage{
			Method: "Connected",
			Data: map[string]string{
				"deviceId":  (*currentSession).DeviceID,
				"sessionId": (*currentSession).ID,
			},
			Timestamp: json.Number(fmt.Sprintf("%d", time.Now().UnixMilli())),
		}

		if err := (*currentSession).SendMessage(confirmMsg); err != nil {
			return fmt.Errorf("failed to send connection confirmation: %w", err)
		}

		broadcastSessionsList()

	case "LatencyPing":
		if *currentSession == nil {
			return fmt.Errorf("no active session for LatencyPing")
		}

		// Handle latency measurement ping
		if data, ok := msg.Data.(map[string]interface{}); ok {
			pongMsg := WSMessage{
				Method: "LatencyPong",
				Data: map[string]interface{}{
					"clientTime":  data["clientTime"],
					"serverTime":  time.Now().UnixMilli(),
					"calibration": data["calibration"],
				},
				Timestamp: json.Number(fmt.Sprintf("%d", time.Now().UnixMilli())),
			}

			if err := (*currentSession).SendMessage(pongMsg); err != nil {
				return fmt.Errorf("failed to send latency pong: %w", err)
			}
		}

	case "Disconnect":
		if *currentSession != nil {
			log.Printf("Explicit disconnect for user %s", (*currentSession).Username)
			sessionManager.RemoveSession(*currentSession)
			broadcastSessionsList()
		}
		return fmt.Errorf("client disconnected")

	case "Pong":
		if *currentSession != nil {
			(*currentSession).UpdatePing()

			// If pong contains timing data, we can use it for server-side latency calculation
			if data, ok := msg.Data.(map[string]interface{}); ok {
				if serverTime, ok := data["serverTime"].(float64); ok {
					if _, ok := data["clientTime"].(float64); ok {
						now := time.Now().UnixMilli()
						rtt := float64(now) - serverTime
						log.Printf("RTT for user %s: %.2fms", (*currentSession).Username, rtt)
					}
				}
			}
		}

	case "UpdateTime":
		if *currentSession == nil {
			return fmt.Errorf("no active session for UpdateTime")
		}

		// Enhanced UpdateTime with platform information
		if data, ok := msg.Data.(map[string]interface{}); ok {
			// Log platform-specific information for debugging
			if platform, ok := data["platform"].(string); ok {
				if audioDelay, ok := data["audioDelay"].(float64); ok {
					log.Printf("UpdateTime from %s platform with %dms audio delay",
						platform, int(audioDelay))
				}
			}
		}

		msg.Timestamp = json.Number(fmt.Sprintf("%d", msg.GetTimestamp()))
		sessionManager.BroadcastToUserExcept((*currentSession).Username, *msg, *currentSession)

	case "SetSong", "Play", "Pause":
		if *currentSession == nil {
			return fmt.Errorf("no active session for %s", msg.Method)
		}

		msg.Timestamp = json.Number(fmt.Sprintf("%d", msg.GetTimestamp()))

		// Add platform context for play/pause commands
		if msg.Method == "Play" || msg.Method == "Pause" {
			log.Printf("Broadcasting %s command from user %s at timestamp %d",
				msg.Method, (*currentSession).Username, msg.GetTimestamp())
		}

		sessionManager.BroadcastToUserExcept((*currentSession).Username, *msg, *currentSession)

	default:
		log.Printf("Unknown message method: %s", msg.Method)
	}

	return nil
}

func broadcastSessionsList() {
	users := sessionManager.GetAllUsers()

	msg := WSMessage{
		Method:    "OtherSessionConnected",
		Data:      users,
		Timestamp: json.Number(fmt.Sprintf("%d", time.Now().UnixMilli())),
	}

	sessionManager.BroadcastExcept(msg, nil)
}
