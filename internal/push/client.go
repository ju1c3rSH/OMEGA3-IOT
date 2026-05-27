package push

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufferSize = 256
)

// Client represents a single WebSocket connection.
type Client struct {
	UserUUID string
	Conn     *websocket.Conn
	SendCh   chan []byte
	mu       sync.Mutex
	closed   bool
}

// NewClient creates a new Client.
func NewClient(userUUID string, conn *websocket.Conn) *Client {
	return &Client{
		UserUUID: userUUID,
		Conn:     conn,
		SendCh:   make(chan []byte, sendBufferSize),
	}
}

// Send marshals a message and queues it for sending.
// Returns false if the send buffer is full (slow client).
func (c *Client) Send(msg *Message) bool {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[PushClient] Failed to marshal message: %v", err)
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}

	select {
	case c.SendCh <- data:
		return true
	default:
		// Buffer full — drop message for this client
		log.Printf("[PushClient] Send buffer full for user %s, dropping message", c.UserUUID)
		return false
	}
}

// Close gracefully closes the client connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.SendCh)
	c.Conn.Close()
}

// ReadPump reads incoming messages from the WebSocket connection.
// It runs in its own goroutine and calls the handler for each message.
func (c *Client) ReadPump(handler MessageHandler) {
	defer func() {
		handler.OnDisconnect(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[PushClient] Read error for user %s: %v", c.UserUUID, err)
			}
			break
		}

		var incoming IncomingMessage
		if err := json.Unmarshal(message, &incoming); err != nil {
			log.Printf("[PushClient] Invalid message from user %s: %v", c.UserUUID, err)
			continue
		}

		handler.OnMessage(c, &incoming)
	}
}

// WritePump writes outgoing messages to the WebSocket connection.
// It runs in its own goroutine and handles ping/pong keepalive.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.SendCh:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages into the same write (batch optimization)
			n := len(c.SendCh)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.SendCh)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// MessageHandler is the interface for handling incoming WebSocket messages.
type MessageHandler interface {
	OnMessage(client *Client, msg *IncomingMessage)
	OnDisconnect(client *Client)
}
