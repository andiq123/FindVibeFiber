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
	Timestamp int64       `json:"timestamp"`
	Latency   int64       `json:"latency,omitempty"`
}

type Session struct {
	Username string
	Conn     *websocket.Conn
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
				log.Println("read error:", err)
				break
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Println("unmarshal error:", err)
				continue
			}

			handleMessage(c, &wsMsg, &username)
		}
	}
}

func cleanupSession(c *websocket.Conn, username string) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	if username == "" {
		return
	}

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
		broadcastSessions()
	}
	c.Close()
}

func handleMessage(c *websocket.Conn, msg *WSMessage, username *string) {
	switch msg.Method {
	case "Connect":
		if uname, ok := msg.Data.(string); ok {
			sessionsMutex.Lock()
			*username = uname
			sessions[uname] = append(sessions[uname], &Session{
				Username: uname,
				Conn:     c,
			})
			broadcastSessions()
			sessionsMutex.Unlock()
		}

	case "Disconnect":
		cleanupSession(c, *username)
		return

	case "UpdateTime":
		switch data := msg.Data.(type) {
		case map[string]interface{}:
			if clientTime, ok := data["clientTime"].(float64); ok {
				now := time.Now().UnixMilli()
				msg.Timestamp = now
				msg.Latency = now - int64(clientTime)
			}
		case string:
			log.Printf("UpdateTime received string data: %s", data)
		default:
			log.Printf("UpdateTime received unexpected data type: %T", msg.Data)
		}
		broadcastToUser(*username, *msg, c)

	case "SetSong", "Play", "Pause":
		broadcastToUser(*username, *msg, c)
	}
}

func broadcastSessions() {
	users := make([]string, 0, len(sessions))
	for username := range sessions {
		users = append(users, username)
	}

	broadcast(WSMessage{
		Method:    "OtherSessionConnected",
		Data:      users,
		Timestamp: time.Now().UnixMilli(),
	})
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

func broadcastToUser(username string, msg WSMessage, senderConn *websocket.Conn) {
	if userSessions, exists := sessions[username]; exists {
		for _, sess := range userSessions {
			if sess.Conn == senderConn {
				continue
			}
			if err := sess.Conn.WriteJSON(msg); err != nil {
				log.Printf("broadcast error to %s: %v", sess.Username, err)
			}
		}
	}
}
