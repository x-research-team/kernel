// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package component

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
	"github.com/x-research-team/utils/is"
	"github.com/x-research-team/utils/magic"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	trunk *contract.ISignalBus
	tcp   *chan []byte
}

func newHub(trunk *contract.ISignalBus, tcp *chan []byte) *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		trunk:      trunk,
		tcp:        tcp,
	}
}

func (h *Hub) run() {
	go h.listen()
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			if !is.JSON(string(message)) {
				bus.Error <- fmt.Errorf("error: received message (%s) is not JSON", message)
				continue
			}
			km := new(KernelMessage)
			if err := json.Unmarshal(message, km); err != nil {
				bus.Error <- err
				continue
			}
			msg := bus.Message(km.Route, km.Command, string(km.Message))
			*h.trunk <- bus.Signal(msg)
		}
	}
}

func (h *Hub) listen() {
	for {
		select {
		case response := <-*h.tcp:
			messages := make([]JournalMessage, 0)
			if !is.JSON(string(response)) {
				bus.Info <- string(response)
				continue
			}
			err := json.Unmarshal(response, &messages)
			switch {
			case err != nil:
				if err := h.fail(err); err != nil {
					break
				}
			case len(messages) == 0:
				if err := h.fail(errors.New("empty response")); err != nil {
					break
				}
			case len(messages) == 1:
				m := messages[0]
				if m.Data == "" {
					continue
				}
				data, err := magic.Jsonify(m.Data)
				if err != nil {
					if err := h.fail(err); err != nil {
						break
					}
					break
				}
				result := JournalMessageResponse{m.ID, data}
				if err := h.send(result); err != nil {
					break
				}
			case len(messages) > 1:
				response := make(JournalMessagesResponse, 0)
				for _, m := range messages {
					if m.Data == "" {
						continue
					}
					data, err := magic.Jsonify(m.Data)
					if err != nil {
						if err := h.fail(err); err != nil {
							break
						}
						break
					}
					result := JournalMessageResponse{m.ID, data}
					response = append(response, result)
				}
				bus.Debug <- response
				if err := h.send(response); err != nil {
					break
				}
			default:
				if err := h.fail(errors.New("bad request")); err != nil {
					break
				}
			}
		default:
			continue
		}
	}
}

func (h *Hub) fail(e error) error {
	if err := h.send(map[string]string{"error": e.Error()}); err != nil {
		bus.Error <- err
		return err
	}
	return nil
}

func (h *Hub) send(v interface{}) error {
	var (
		err    error
		buffer []byte
	)
	if buffer, err = json.Marshal(v); err != nil {
		bus.Error <- err
		return err
	}
	for client := range h.clients {
		select {
		case client.send <- buffer:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
	return nil
}
