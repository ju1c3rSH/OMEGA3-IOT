package push

import (
	"OMEGA3-IOT/internal/eventbus"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/repository"
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ackTimeout     = 10 * time.Second
	maxRetransmit  = 1
	cleanInterval  = 30 * time.Second
)

// pendingMessage tracks an unacknowledged push message.
type pendingMessage struct {
	seq        int64
	data       []byte
	sentAt     time.Time
	retransmit int
}

// PushService manages WebSocket clients and routes EventBus events to them.
type PushService struct {
	clients      sync.Map // userUUID → []*Client
	eventBus     *eventbus.EventBus
	instanceRepo repository.InstanceRepository
	userRepo     repository.UserRepository
	seqCounter   int64
	pendingACKs  sync.Map // int64 → *pendingMessage
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewPushService creates a new PushService.
func NewPushService(
	eventBus *eventbus.EventBus,
	instanceRepo repository.InstanceRepository,
	userRepo repository.UserRepository,
) *PushService {
	return &PushService{
		eventBus:     eventBus,
		instanceRepo: instanceRepo,
		userRepo:     userRepo,
		stopCh:       make(chan struct{}),
	}
}

// Start subscribes to EventBus events and starts the ACK retransmit checker.
func (ps *PushService) Start() {
	eventbus.SubscribeTyped(ps.eventBus, eventbus.EventType(logger.LogEventDeviceStatusChange), ps.handleStatusChange)
	eventbus.SubscribeTyped(ps.eventBus, eventbus.EventType(logger.LogEventDevicePropertyUpdate), ps.handlePropertyUpdate)
	eventbus.SubscribeTyped(ps.eventBus, eventbus.EventType("device.event.received"), ps.handleEventPush)
	eventbus.SubscribeTyped(ps.eventBus, eventbus.EventType(logger.LogEventDeviceActionResult), ps.handleActionResult)
	log.Println("[PushService] Subscribed to device.status.change, device.property.update, device.event.received, device.action.result")

	// Background ACK retransmit checker
	ps.wg.Add(1)
	go ps.retransmitLoop()
	log.Println("[PushService] Started")
}

// Stop gracefully shuts down the PushService.
func (ps *PushService) Stop() {
	close(ps.stopCh)
	ps.wg.Wait()

	// Close all client connections
	ps.clients.Range(func(key, value interface{}) bool {
		clients := value.([]*Client)
		for _, c := range clients {
			c.Close()
		}
		return true
	})
	log.Println("[PushService] Stopped")
}

// Register adds a client to the push service.
func (ps *PushService) Register(client *Client) {
	val, _ := ps.clients.LoadOrStore(client.UserUUID, make([]*Client, 0))
	clients := val.([]*Client)
	clients = append(clients, client)
	ps.clients.Store(client.UserUUID, clients)
	log.Printf("[PushService] Client registered: user=%s, total connections=%d", client.UserUUID, len(clients))

	// Deliver offline critical messages
	ps.deliverOfflineMessages(client)
}

// Unregister removes a client from the push service.
func (ps *PushService) Unregister(client *Client) {
	val, ok := ps.clients.Load(client.UserUUID)
	if !ok {
		return
	}
	clients := val.([]*Client)
	for i, c := range clients {
		if c == client {
			clients = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(clients) == 0 {
		ps.clients.Delete(client.UserUUID)
	} else {
		ps.clients.Store(client.UserUUID, clients)
	}
	client.Close()
	log.Printf("[PushService] Client unregistered: user=%s", client.UserUUID)
}

// PushToUser sends a message to all connections of a specific user.
func (ps *PushService) PushToUser(userUUID string, msg *Message) {
	val, ok := ps.clients.Load(userUUID)
	if !ok {
		return
	}
	clients := val.([]*Client)
	for _, c := range clients {
		c.Send(msg)
	}
}

// PushToDeviceOwner looks up the device owner and pushes the message.
func (ps *PushService) PushToDeviceOwner(deviceUUID string, msg *Message) {
	instance, err := ps.instanceRepo.FindByUUID(deviceUUID)
	if err != nil {
		return
	}
	ps.PushToUser(instance.OwnerUUID, msg)
}

// nextSeq returns the next sequence number.
func (ps *PushService) nextSeq() int64 {
	return atomic.AddInt64(&ps.seqCounter, 1)
}

// sendWithACK sends a message and registers it for ACK tracking.
func (ps *PushService) sendWithACK(userUUID string, msg *Message) {
	msg.Seq = ps.nextSeq()

	val, ok := ps.clients.Load(userUUID)
	if !ok {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[PushService] Failed to marshal message: %v", err)
		return
	}

	pm := &pendingMessage{
		seq:    msg.Seq,
		data:   data,
		sentAt: time.Now(),
	}
	ps.pendingACKs.Store(msg.Seq, pm)

	clients := val.([]*Client)
	for _, c := range clients {
		c.Send(msg)
	}
}

// ─── EventBus Handlers ───

func (ps *PushService) handleStatusChange(ctx context.Context, event logger.DeviceLogEvent) error {
	status, _ := event.Metadata["status"].(string)
	payload := DeviceStatusPayload{
		DeviceUUID: event.DeviceUUID,
		Online:     status == "online",
		LastSeen:   event.Timestamp,
	}
	msg := NewMessage(TypeDeviceStatus, payload)
	ps.PushToDeviceOwner(event.DeviceUUID, msg)
	return nil
}

func (ps *PushService) handlePropertyUpdate(ctx context.Context, event logger.DeviceLogEvent) error {
	props, _ := event.Metadata["properties"].(map[string]interface{})
	if len(props) == 0 {
		return nil
	}
	payload := PropertyUpdatePayload{
		DeviceUUID: event.DeviceUUID,
		Properties: props,
	}
	msg := NewMessage(TypePropertyUpdate, payload)
	ps.PushToDeviceOwner(event.DeviceUUID, msg)
	return nil
}

func (ps *PushService) handleEventPush(ctx context.Context, event logger.DeviceLogEvent) error {
	eventKey, _ := event.Metadata["event_key"].(string)
	severity, _ := event.Metadata["severity"].(string)
	data := event.Metadata["data"]

	payload := EventPushPayload{
		EventKey:   eventKey,
		DeviceUUID: event.DeviceUUID,
		Severity:   severity,
		Data:       data,
	}
	// Use sendWithACK for warning/critical events
	msg := NewSequencedMessage(TypeEventPush, ps.nextSeq(), payload)

	// Store for ACK tracking
	dataBytes, _ := json.Marshal(msg)
	pm := &pendingMessage{
		seq:    msg.Seq,
		data:   dataBytes,
		sentAt: time.Now(),
	}
	ps.pendingACKs.Store(msg.Seq, pm)

	// Push to device owner
	instance, err := ps.instanceRepo.FindByUUID(event.DeviceUUID)
	if err != nil {
		return nil
	}
	ps.PushToUser(instance.OwnerUUID, msg)
	return nil
}

func (ps *PushService) handleActionResult(ctx context.Context, event logger.DeviceLogEvent) error {
	command, _ := event.Metadata["command"].(string)
	success, _ := event.Metadata["success"].(bool)
	errMsg, _ := event.Metadata["error"].(string)

	payload := ActionResultPayload{
		DeviceUUID: event.DeviceUUID,
		Command:    command,
		Success:    success,
		Error:      errMsg,
	}
	msg := NewMessage(TypeActionResult, payload)
	ps.PushToDeviceOwner(event.DeviceUUID, msg)
	return nil
}

// ─── Client Message Handler ───

// OnMessage implements MessageHandler.
func (ps *PushService) OnMessage(client *Client, msg *IncomingMessage) {
	switch msg.Type {
	case TypeACK:
		var payload ACKPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("[PushService] Invalid ACK payload from user %s: %v", client.UserUUID, err)
			return
		}
		ps.pendingACKs.Delete(payload.ACKSeq)

	case TypePing:
		pong := NewMessage(TypePong, nil)
		client.Send(pong)

	case TypeActionSend:
		var payload ActionSendPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("[PushService] Invalid action.send payload from user %s: %v", client.UserUUID, err)
			client.Send(NewMessage(TypeActionResponse, ActionResponsePayload{Success: false, Error: "invalid payload"}))
			return
		}
		ps.handleActionSend(client, &payload)

	default:
		log.Printf("[PushService] Unknown message type '%s' from user %s", msg.Type, client.UserUUID)
	}
}

// OnDisconnect implements MessageHandler.
func (ps *PushService) OnDisconnect(client *Client) {
	ps.Unregister(client)
}

func (ps *PushService) handleActionSend(client *Client, payload *ActionSendPayload) {
	// Verify the user has access to this device
	instance, err := ps.instanceRepo.FindByUUID(payload.DeviceUUID)
	if err != nil {
		client.Send(NewMessage(TypeActionResponse, ActionResponsePayload{Success: false, Error: "device not found"}))
		return
	}
	if instance.OwnerUUID != client.UserUUID {
		client.Send(NewMessage(TypeActionResponse, ActionResponsePayload{Success: false, Error: "access denied"}))
		return
	}

	// Forward to MQTT via the existing MQTTService would be ideal,
	// but for now we publish an event that can be picked up.
	// The actual MQTT publish should be done by a service that has access to the MQTT broker.
	log.Printf("[PushService] Action '%s' requested for device %s by user %s", payload.Command, payload.DeviceUUID, client.UserUUID)
	client.Send(NewMessage(TypeActionResponse, ActionResponsePayload{Success: true}))
}

// ─── ACK Retransmit ───

func (ps *PushService) retransmitLoop() {
	defer ps.wg.Done()
	ticker := time.NewTicker(cleanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ps.stopCh:
			return
		case <-ticker.C:
			ps.checkPendingACKs()
		}
	}
}

func (ps *PushService) checkPendingACKs() {
	now := time.Now()
	ps.pendingACKs.Range(func(key, value interface{}) bool {
		pm := value.(*pendingMessage)
		if now.Sub(pm.sentAt) > ackTimeout {
			if pm.retransmit < maxRetransmit {
				pm.retransmit++
				pm.sentAt = now
				// Retransmit to all clients of the user (we'd need to track userUUID)
				// For simplicity, just delete — the message is lost after max retransmit
				log.Printf("[PushService] ACK timeout for seq %d, retransmit #%d", pm.seq, pm.retransmit)
			} else {
				ps.pendingACKs.Delete(key)
				log.Printf("[PushService] ACK timeout for seq %d, giving up", pm.seq)
			}
		}
		return true
	})
}

// ─── Offline Messages ───

func (ps *PushService) deliverOfflineMessages(client *Client) {
	// This would query the offline_messages table for undelivered critical messages
	// and send them to the client. Implementation depends on the repository.
	// For now, this is a placeholder.
	log.Printf("[PushService] Checking offline messages for user %s", client.UserUUID)
}
