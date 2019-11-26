// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package bus

import (
    "go-bifrost/stompserver"
    "go-bifrost/model"
    "encoding/json"
    "go-bifrost/log"
    "github.com/go-stomp/stomp/frame"
    "strings"
    "sync"
    "fmt"
)

type EndpointConfig struct {
    // Prefix for public topics e.g. "/topic"
    TopicPrefix           string
    // Prefix for user queues e.g. "/user/queue"
    UserQueuePrefix       string
    // Prefix used for public application requests e.g. "/pub"
    AppRequestPrefix      string
    // Prefix used for "private" application requests e.g. "/pub/queue"
    // Requests sent to destinations prefixed with the AppRequestQueuePrefix
    // should generate responses sent to single client queue.
    // E.g. if a client sends a request to the "/pub/queue/sample-channel" destination
    // the application should sent the response only to this client on the
    // "/user/queue/sample-channel" destination.
    // This behavior will mimic the Spring SimpleMessageBroker implementation.
    AppRequestQueuePrefix string
    Heartbeat             int64
}

func (ec *EndpointConfig) validate() error {
    if ec.TopicPrefix == "" || !strings.HasPrefix(ec.TopicPrefix, "/") {
        return fmt.Errorf("invalid TopicPrefix")
    }

    if ec.AppRequestQueuePrefix != "" && ec.UserQueuePrefix == "" {
        return fmt.Errorf("missing UserQueuePrefix")
    }

    return nil
}

type FabricEndpoint interface {
    Start()
    Stop()
}

type channelMapping struct {
    subs        map[string]bool
    handler     MessageHandler
    autoCreated bool
}

type fabricEndpoint struct {
    server stompserver.StompServer
    bus    EventBus
    config EndpointConfig
    chanLock sync.RWMutex
    chanMappings map[string]*channelMapping
}

func addPrefixIfNotEmpty(s string, prefix string) string {
    if s != "" && !strings.HasSuffix(s, prefix) {
        return s + prefix
    }
    return s
}

func newFabricEndpoint(bus EventBus,
        conListener stompserver.RawConnectionListener, config EndpointConfig) FabricEndpoint {

    config.TopicPrefix = addPrefixIfNotEmpty(config.TopicPrefix, "/")
    config.AppRequestPrefix = addPrefixIfNotEmpty(config.AppRequestPrefix, "/")
    config.AppRequestQueuePrefix = addPrefixIfNotEmpty(config.AppRequestQueuePrefix, "/")
    config.UserQueuePrefix = addPrefixIfNotEmpty(config.UserQueuePrefix, "/")

    stompConf := stompserver.NewStompConfig(config.Heartbeat,
            []string{config.AppRequestPrefix, config.AppRequestQueuePrefix})

    fabricEndpoint := &fabricEndpoint{
        server:       stompserver.NewStompServer(conListener, stompConf),
        config:       config,
        bus:          bus,
        chanMappings: make(map[string]*channelMapping),
    }

    fabricEndpoint.initHandlers()
    return fabricEndpoint
}

func (fe *fabricEndpoint) Start() {
    fe.server.Start()
}

func (fe *fabricEndpoint) Stop() {
    fe.server.Stop()
}

func (fe *fabricEndpoint) initHandlers() {
    fe.server.OnApplicationRequest(fe.bridgeMessage)
    fe.server.OnSubscribeEvent(fe.addSubscription)
    fe.server.OnUnsubscribeEvent(fe.removeSubscription)
}

func (fe *fabricEndpoint) addSubscription(
        conId string, subId string, destination string, frame *frame.Frame) {

    channelName, ok := fe.getChannelNameFromSubscription(destination)
    if !ok {
        return
    }

    fe.chanLock.Lock()
    defer fe.chanLock.Unlock()

    chanMap, ok := fe.chanMappings[channelName]
    if !ok {
        messageHandler, err := fe.bus.ListenStream(channelName)
        var autoCreated = false
        if messageHandler == nil || err != nil {
            fe.bus.GetChannelManager().CreateChannel(channelName)
            messageHandler, err = fe.bus.ListenStream(channelName)
            if messageHandler == nil || err != nil {
                log.Warn("Unable to auto-create channel for destination: %s", destination)
                return
            }
            autoCreated = true
        }
        messageHandler.Handle(
            func(message *model.Message) {
                data, err := marshalMessagePayload(message)
                if err == nil {
                    resp, ok := convertPayloadToResponseObj(message)
                    if ok && resp != nil && resp.BrokerDestination != nil {
                        fe.server.SendMessageToClient(
                            resp.BrokerDestination.ConnectionId,
                            resp.BrokerDestination.Destination,
                            data)
                    } else {
                        fe.server.SendMessage(fe.config.TopicPrefix + channelName, data)
                    }
                }
            },
            func(e error) {
                fe.server.SendMessage(destination, []byte(e.Error()))
            })

        chanMap = &channelMapping{
            subs:    make(map[string]bool),
            handler: messageHandler,
            autoCreated: autoCreated,
        }

        fe.chanMappings[channelName] = chanMap
    }
    chanMap.subs[conId + "#" + subId] = true
    fe.bus.SendMonitorEvent(FabricEndpointSubscribeEvt, channelName, nil)
}

func convertPayloadToResponseObj(message *model.Message) (*model.Response, bool) {
    var resp model.Response
    var ok bool

    resp, ok = message.Payload.(model.Response)
    if ok {
        return &resp, true
    }

    var respPtr *model.Response
    respPtr, ok = message.Payload.(*model.Response)
    if ok {
        return respPtr, true
    }

    return nil, false
}

func marshalMessagePayload(message *model.Message) ([]byte, error) {
    // don't marshal string and []byte payloads
    stringPayload, ok := message.Payload.(string)
    if ok {
        return []byte(stringPayload), nil
    }
    bytePayload, ok := message.Payload.([]byte)
    if ok {
        return bytePayload, nil
    }
    // encode the message payload as JSON
    return json.Marshal(message.Payload)
}

func (fe *fabricEndpoint) removeSubscription(conId string, subId string, destination string) {

    channelName, ok := fe.getChannelNameFromSubscription(destination)
    if !ok {
        return
    }

    fe.chanLock.Lock()
    defer fe.chanLock.Unlock()

    chanMap, ok := fe.chanMappings[channelName]
    if ok {
        mappingId := conId + "#" + subId
        if chanMap.subs[mappingId] {
            delete(chanMap.subs, mappingId)
            if len(chanMap.subs) == 0 {
                // if this was the last subscription to the channel,
                // close the message handler and remove the channel mapping
                chanMap.handler.Close()
                delete(fe.chanMappings, channelName)
                if chanMap.autoCreated {
                    fe.bus.GetChannelManager().DestroyChannel(channelName)
                }
            }
            fe.bus.SendMonitorEvent(FabricEndpointUnsubscribeEvt, channelName, nil)
        }
    }
}

func (fe *fabricEndpoint) bridgeMessage(destination string, message []byte, connectionId string) {
    var channelName string
    isPrivateRequest := false

    if fe.config.AppRequestQueuePrefix != "" && strings.HasPrefix(destination, fe.config.AppRequestQueuePrefix) {
        channelName = destination[len(fe.config.AppRequestQueuePrefix):]
        isPrivateRequest = true
    } else if fe.config.AppRequestPrefix != "" && strings.HasPrefix(destination, fe.config.AppRequestPrefix) {
        channelName = destination[len(fe.config.AppRequestPrefix):]
    } else {
        return
    }

    var req model.Request
    err := json.Unmarshal(message, &req)
    if err != nil {
        log.Warn("Failed to deserialize request for channel %s", channelName)
        return
    }

    if isPrivateRequest {
        req.BrokerDestination = &model.BrokerDestinationConfig{
            Destination: fe.config.UserQueuePrefix + channelName,
            ConnectionId: connectionId,
        }
    }

    fe.bus.SendRequestMessage(channelName, &req, nil)
}

func (fe *fabricEndpoint) getChannelNameFromSubscription(destination string) (channelName string, ok bool) {
    if strings.HasPrefix(destination, fe.config.TopicPrefix) {
        return destination[len(fe.config.TopicPrefix):], true
    }

    if fe.config.UserQueuePrefix != "" && strings.HasPrefix(destination, fe.config.UserQueuePrefix) {
        return destination[len(fe.config.UserQueuePrefix):], true
    }
    return "", false
}
