package handlers

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"slices"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type WSMessage struct {
	Method    string      `json:"method"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

type Session struct {
	Username string
	Conn     *websocket.Conn
	Closed   bool
	DeviceID string
}

var (
	sessions      = make(map[string][]*Session)
	sessionsMutex sync.RWMutex
)

func PlayerWebSocketHandler() func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		var username string
		defer cleanupSession(c, username)

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Printf("read error for user %s: %v", username, err)
				break
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Printf("unmarshal error for user %s: %v", username, err)
				continue
			}

			handleMessage(c, &wsMsg, &username)
		}
	}
}

func cleanupSession(c *websocket.Conn, username string) {
	if username == "" {
		return
	}

	var shouldBroadcast bool
	var remainingSessions int

	func() {
		sessionsMutex.Lock()
		defer sessionsMutex.Unlock()

		log.Printf("Cleaning up session for user: %s", username)
		if userSessions, exists := sessions[username]; exists {
			// Find and remove this specific session
			for i, sess := range userSessions {
				if sess.Conn == c {
					log.Printf("Removing device %s for user %s", sess.DeviceID, username)
					sess.Closed = true
					sessions[username] = slices.Delete(userSessions, i, i+1)
					break
				}
			}

			// If no more sessions for this user, remove the user
			if len(sessions[username]) == 0 {
				log.Printf("No more sessions for user %s, removing user", username)
				delete(sessions, username)
			} else {
				remainingSessions = len(sessions[username])
				log.Printf("User %s has %d remaining sessions", username, remainingSessions)
			}
			shouldBroadcast = true
		}
	}()

	c.Close()

	if shouldBroadcast {
		broadcastSessionsExcept(c)
	}
}

func handleMessage(c *websocket.Conn, msg *WSMessage, username *string) {
	switch msg.Method {
	case "Connect":
		if uname, ok := msg.Data.(string); ok {
			deviceID := uuid.New().String()
			func() {
				sessionsMutex.Lock()
				defer sessionsMutex.Unlock()
				*username = uname
				sessions[uname] = append(sessions[uname], &Session{
					Username: uname,
					Conn:     c,
					Closed:   false,
					DeviceID: deviceID,
				})
				log.Printf("New device %s connected for user %s", deviceID, uname)
			}()
			// Broadcast to all other users except the sender
			broadcastSessionsExcept(c)
		}

	case "Disconnect":
		log.Printf("Disconnect event received for user %s", *username)
		cleanupSession(c, *username)
		return

	case "UpdateTime":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if eventTime, ok := data["timestamp"].(float64); ok {
				msg.Timestamp = int64(eventTime)
			}
		}
		broadcastToUserExcept(*username, *msg, c)

	case "SetSong", "Play", "Pause":
		broadcastToUserExcept(*username, *msg, c)
	}
}

func broadcastSessionsExcept(senderConn *websocket.Conn) {
	var users []string
	func() {
		sessionsMutex.RLock()
		defer sessionsMutex.RUnlock()
		users = make([]string, 0, len(sessions))
		for username := range sessions {
			users = append(users, username)
		}
	}()

	broadcastExcept(WSMessage{
		Method:    "OtherSessionConnected",
		Data:      users,
		Timestamp: time.Now().UnixMilli(),
	}, senderConn)
}

func broadcastExcept(msg WSMessage, senderConn *websocket.Conn) {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	for _, userSessions := range sessions {
		for _, sess := range userSessions {
			if !sess.Closed && sess.Conn != senderConn {
				if err := sess.Conn.WriteJSON(msg); err != nil {
					log.Printf("broadcast error to %s (device %s): %v", sess.Username, sess.DeviceID, err)
					sess.Closed = true
				}
			}
		}
	}
}

func broadcastToUserExcept(username string, msg WSMessage, senderConn *websocket.Conn) {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	if userSessions, exists := sessions[username]; exists {
		for _, sess := range userSessions {
			if sess.Conn == senderConn || sess.Closed {
				continue
			}
			if err := sess.Conn.WriteJSON(msg); err != nil {
				log.Printf("broadcast error to %s (device %s): %v", sess.Username, sess.DeviceID, err)
				sess.Closed = true
			}
		}
	}
}
