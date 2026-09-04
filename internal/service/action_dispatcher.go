package service

import (
	"OMEGA3-IOT/internal/eventbus"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/model"
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

// actionDispatchMaxConcurrent bounds in-flight MQTT publishes. When the
// broker is degraded, PublishActionToDevice blocks up to mqttPublishTimeout;
// the semaphore keeps the number of blocked goroutines bounded so callers
// (HTTP handler, WebSocket handler) only wait for a free slot.
const actionDispatchMaxConcurrent = 64

// ActionPublisher is the minimal MQTT-facing contract ActionDispatcher needs.
// *MQTTService satisfies it.
type ActionPublisher interface {
	PublishActionToDevice(deviceUUID string, commandName string, payload model.Action) error
}

// ActionDispatcher publishes device actions asynchronously so callers are not
// blocked by broker latency. Failures are surfaced as device.error events on
// the EventBus instead of synchronous errors.
type ActionDispatcher struct {
	publisher ActionPublisher
	eventBus  *eventbus.EventBus
	sem       chan struct{}
	wg        sync.WaitGroup
}

func NewActionDispatcher(publisher ActionPublisher, eventBus *eventbus.EventBus) *ActionDispatcher {
	return &ActionDispatcher{
		publisher: publisher,
		eventBus:  eventBus,
		sem:       make(chan struct{}, actionDispatchMaxConcurrent),
	}
}

// Dispatch hands the action off to a background goroutine and returns
// immediately. Payload should carry a unique ActionID for result correlation.
func (d *ActionDispatcher) Dispatch(deviceUUID, commandName string, payload model.Action) {
	d.wg.Add(1)
	d.sem <- struct{}{}
	go func() {
		defer func() {
			<-d.sem
			if r := recover(); r != nil {
				log.Printf("[ActionDispatcher] panic dispatching command=%s action_id=%s device=%s: %v\n%s",
					commandName, payload.ActionID, deviceUUID, r, debug.Stack())
				d.publishErrorEvent(deviceUUID, commandName, payload.ActionID, fmt.Sprintf("internal panic: %v", r))
			}
			d.wg.Done()
		}()
		if err := d.publisher.PublishActionToDevice(deviceUUID, commandName, payload); err != nil {
			log.Printf("[ActionDispatcher] failed to publish command=%s action_id=%s device=%s: %v",
				commandName, payload.ActionID, deviceUUID, err)
			d.publishErrorEvent(deviceUUID, commandName, payload.ActionID, err.Error())
		}
	}()
}

func (d *ActionDispatcher) publishErrorEvent(deviceUUID, commandName, actionID, errMsg string) {
	if d.eventBus == nil {
		return
	}
	errEvent := logger.NewDeviceLogEvent(deviceUUID, logger.LogLevelWarning,
		fmt.Sprintf("Action dispatch failed: %s", commandName), logger.LogEventDeviceError)
	errEvent.Metadata["command"] = commandName
	errEvent.Metadata["action_id"] = actionID
	errEvent.Metadata["error"] = errMsg
	d.eventBus.Publish(context.Background(), errEvent)
}

// Stop waits for all in-flight dispatches to finish.
func (d *ActionDispatcher) Stop() {
	d.wg.Wait()
	log.Println("[ActionDispatcher] stopped, all in-flight dispatches drained")
}
