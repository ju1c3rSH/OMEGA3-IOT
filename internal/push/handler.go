package push

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: restrict origin in production
		return true
	},
}

// PushHandler handles the WebSocket upgrade endpoint.
type PushHandler struct {
	pushService *PushService
}

// NewPushHandler creates a new PushHandler.
func NewPushHandler(pushService *PushService) *PushHandler {
	return &PushHandler{pushService: pushService}
}

// HandleWebSocket upgrades the HTTP connection to WebSocket.
func (h *PushHandler) HandleWebSocket(c *gin.Context) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[PushHandler] WebSocket upgrade failed: %v", err)
		return
	}

	client := NewClient(userUUID.(string), conn)
	h.pushService.Register(client)

	// Both pumps run in separate goroutines so the Gin handler returns immediately.
	// ReadPump's defer handles disconnect cleanup (Unregister + Close).
	go client.WritePump()
	go client.ReadPump(h.pushService)
}
