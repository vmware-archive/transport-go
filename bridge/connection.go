// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package bridge

import (
	"fmt"
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/google/uuid"
	"github.com/vmware/transport-go/model"
	"log"
	"sync"
)

type Connection interface {
	GetId() *uuid.UUID
	Subscribe(destination string) (Subscription, error)
	SubscribeReplyDestination(destination string) (Subscription, error)
	Disconnect() (err error)
	SendJSONMessage(destination string, payload []byte, opts ...func(*frame.Frame) error) error
	SendMessage(destination, contentType string, payload []byte, opts ...func(*frame.Frame) error) error
	SendMessageWithReplyDestination(destination, replyDestination, contentType string, payload []byte, opts ...func(*frame.Frame) error) error
}

// Connection represents a Connection to a message broker.
type connection struct {
	id             *uuid.UUID
	useWs          bool
	conn           *stomp.Conn
	wsConn         *BridgeClient
	disconnectChan chan bool
	subscriptions  map[string]Subscription
	connLock       sync.Mutex
}

func (c *connection) GetId() *uuid.UUID {
	return c.id
}

// Subscribe to a destination, only one subscription can exist for a destination
func (c *connection) Subscribe(destination string) (Subscription, error) {
	// check if the subscription exists, if so, return it.
	if c == nil {
		return nil, fmt.Errorf("cannot subscribe to '%s', no connection to broker", destination)
	}

	c.connLock.Lock()
	if sub, ok := c.subscriptions[destination]; ok {
		c.connLock.Unlock()
		return sub, nil
	}
	c.connLock.Unlock()

	// use websocket, if not use stomp TCP.
	if c.useWs {
		return c.subscribeWs(destination)
	}

	return c.subscribeTCP(destination)
}

// SubscribeReplyDestination subscribe to a reply destination (this will create an internal subscription to the
// destination, but won't actually send the request over the wire to the broker. This is because when using temp
// queues, the destinations are dynamic. The raw socket is send responses that are to a destination that
// does not actually exist when using reply-to so this will allow that imaginary destination to operate.
func (c *connection) SubscribeReplyDestination(destination string) (Subscription, error) {
	// check if the subscription exists, if so, return it.
	if c == nil {
		return nil, fmt.Errorf("cannot subscribe to '%s', no connection to broker", destination)
	}

	c.connLock.Lock()
	if sub, ok := c.subscriptions[destination]; ok {
		c.connLock.Unlock()
		return sub, nil
	}
	c.connLock.Unlock()

	// use websocket, if not use stomp TCP.
	if c.useWs {
		return c.subscribeWs(destination)
	}

	return c.subscribeTCPUsingReplyDestination(destination)
}

// Disconnect from broker, will close all channels
func (c *connection) Disconnect() (err error) {
	if c == nil {
		return fmt.Errorf("cannot disconnect, not connected")
	}
	if c.useWs {
		if c.wsConn != nil && c.wsConn.connected {
			defer c.cleanUpConnection()
			err = c.wsConn.Disconnect()
		}
	} else {
		if c.conn != nil {
			defer c.cleanUpConnection()
			err = c.conn.Disconnect()
		}
	}
	return err
}

func (c *connection) cleanUpConnection() {
	if c.conn != nil {
		c.conn = nil
	}
	if c.wsConn != nil {
		c.wsConn = nil
	}
}

func (c *connection) subscribeWs(destination string) (Subscription, error) {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	if c.wsConn != nil {
		wsSub := c.wsConn.Subscribe(destination)
		sub := &subscription{wsStompSub: wsSub, id: wsSub.Id, c: wsSub.C, destination: destination}
		c.subscriptions[destination] = sub
		return sub, nil
	}
	return nil, fmt.Errorf("cannot subscribe, websocket not connected / established")
}

func (c *connection) subscribeTCP(destination string) (Subscription, error) {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	if c.conn != nil {
		sub, _ := c.conn.Subscribe(destination, stomp.AckAuto)
		id := uuid.New()
		destChan := make(chan *model.Message)
		go c.listenTCPFrames(sub.C, destChan)
		bcSub := &subscription{stompTCPSub: sub, id: &id, c: destChan}
		c.subscriptions[destination] = bcSub
		return bcSub, nil
	}
	return nil, fmt.Errorf("no STOMP TCP connection established")
}

func (c *connection) subscribeTCPUsingReplyDestination(destination string) (Subscription, error) {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	if c.conn != nil {

		var reply = func(f *frame.Frame) error {
			f.Header.Add("reply-to", destination)
			return nil
		}

		sub, _ := c.conn.Subscribe(destination, stomp.AckAuto, reply)
		id := uuid.New()
		destChan := make(chan *model.Message)
		go c.listenTCPFrames(sub.C, destChan)
		bcSub := &subscription{stompTCPSub: sub, id: &id, c: destChan}
		c.subscriptions[destination] = bcSub
		return bcSub, nil
	}
	return nil, fmt.Errorf("no STOMP TCP connection established")
}

func (c *connection) listenTCPFrames(src chan *stomp.Message, dst chan *model.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("subscription is closed, message undeliverable to closed channel.")
		}
	}()
	for {
		f := <-src
		var body []byte
		var dest string
		if f != nil && f.Body != nil {
			body = f.Body
		}
		if f != nil && len(f.Destination) > 0 {
			dest = f.Destination
		}
		if f != nil {
			cf := &model.MessageConfig{Payload: body, Destination: dest}

			// transfer over known non-standard, but important frame headers if they are set
			if replyTo, ok := f.Header.Contains("reply-to"); ok { // used by rabbitmq for temp queues
				cf.Headers = []model.MessageHeader{{Label: "reply-to", Value: replyTo}}
			}

			m := model.GenerateResponse(cf)
			dst <- m
		}
	}
}

// SendJSONMessage sends a []byte payload carrying JSON data to a destination.
func (c *connection) SendJSONMessage(destination string, payload []byte, opts ...func(*frame.Frame) error) error {
	return c.SendMessage(destination, "application/json", payload, opts...)
}

// SendMessageWithReplyDestination is the same as SendMessage, but adds in a reply-to header automatically.
// This is generally used in conjunction with SubscribeReplyDestination
func (c *connection) SendMessageWithReplyDestination(destination string, replyDestination, contentType string, payload []byte, opts ...func(*frame.Frame) error) error {
	var headerReplyTo = func(f *frame.Frame) error {
		f.Header.Add("reply-to", replyDestination)
		return nil
	}
	opts = append(opts, headerReplyTo)
	return c.SendMessage(destination, contentType, payload, opts...)
}

// SendMessage will send a []byte payload to a destination.
func (c *connection) SendMessage(destination string, contentType string, payload []byte, opts ...func(*frame.Frame) error) error {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	if c != nil && !c.useWs && c.conn != nil {
		c.conn.Send(destination, contentType, payload, opts...)
		return nil
	}
	if c != nil && c.useWs && c.wsConn != nil {
		c.wsConn.Send(destination, contentType, payload, opts...)
		return nil
	}
	return fmt.Errorf("cannot send message, no connection")

}
