package handlers

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

type WSMessage struct {
	Method    string      `json:"method"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"` // Unix timestamp in milliseconds
}

type Session struct {
	Username string
	Conn     *websocket.Conn
	Latency  int64 // Round-trip latency in milliseconds
}

var (
	sessions      = make(map[string][]*Session)
	sessionsMutex sync.Mutex
)

func PlayerWebSocketHandler() func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		var username string
		defer func() {
			sessionsMutex.Lock()
			if username != "" {
				if userSessions, exists := sessions[username]; exists {
					for i, sess := range userSessions {
						if sess.Conn == c {
							sessions[username] = append(userSessions[:i], userSessions[i+1:]...)
							break
						}
					}
					if len(sessions[username]) == 0 {
						delete(sessions, username)
					}
				}
				broadcastSessions()
			}
			sessionsMutex.Unlock()
			c.Close()
		}()

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				break
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Println("unmarshal error:", err)
				continue
			}

			// Add server timestamp to the message
			wsMsg.Timestamp = time.Now().UnixMilli()

			switch wsMsg.Method {
			case "Connect":
				uname, ok := wsMsg.Data.(string)
				if !ok {
					continue
				}
				sessionsMutex.Lock()
				username = uname
				sessions[username] = append(sessions[username], &Session{
					Username: username,
					Conn:     c,
					Latency:  0,
				})
				broadcastSessions()
				sessionsMutex.Unlock()

			case "Ping":
				// Handle latency measurement
				if pingData, ok := wsMsg.Data.(map[string]interface{}); ok {
					if clientTime, ok := pingData["clientTime"].(float64); ok {
						latency := time.Now().UnixMilli() - int64(clientTime)
						sessionsMutex.Lock()
						if userSessions, exists := sessions[username]; exists {
							for _, sess := range userSessions {
								if sess.Conn == c {
									sess.Latency = latency
									break
								}
							}
						}
						sessionsMutex.Unlock()
					}
				}
				// Send pong response
				c.WriteJSON(WSMessage{
					Method:    "Pong",
					Timestamp: time.Now().UnixMilli(),
				})

			case "Disconnect":
				sessionsMutex.Lock()
				if username != "" {
					if userSessions, exists := sessions[username]; exists {
						for i, sess := range userSessions {
							if sess.Conn == c {
								sessions[username] = append(userSessions[:i], userSessions[i+1:]...)
								break
							}
						}
						if len(sessions[username]) == 0 {
							delete(sessions, username)
						}
					}
					broadcastSessions()
				}
				sessionsMutex.Unlock()
				return

			case "SetSong":
				broadcastToUser(username, wsMsg, c)

			case "Play":
				broadcastToUser(username, wsMsg, c)

			case "Pause":
				broadcastToUser(username, wsMsg, c)

			case "UpdateTime":
				broadcastToUser(username, wsMsg, c)
			}
		}
	}
}

func broadcastSessions() {
	uniqueUsers := make([]string, 0)
	for username := range sessions {
		uniqueUsers = append(uniqueUsers, username)
	}
	msg := WSMessage{
		Method:    "OtherSessionConnected",
		Data:      uniqueUsers,
		Timestamp: time.Now().UnixMilli(),
	}
	broadcast(msg)
}

func broadcast(msg WSMessage) {
	for _, userSessions := range sessions {
		for _, sess := range userSessions {
			if err := sess.Conn.WriteJSON(msg); err != nil {
				log.Printf("broadcast error to %s: %v", sess.Username, err)
			}
		}
	}
}

func broadcastToUser(username string, msg WSMessage, excludeConn *websocket.Conn) {
	if userSessions, exists := sessions[username]; exists {
		for _, sess := range userSessions {
			if sess.Conn != excludeConn {
				if err := sess.Conn.WriteJSON(msg); err != nil {
					log.Printf("broadcast error to %s: %v", sess.Username, err)
				}
			}
		}
	}
}
