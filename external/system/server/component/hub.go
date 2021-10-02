/*
 *   Copyright (c) 2021 Adel Urazov
 *   All rights reserved.

 *   Permission is hereby granted, free of charge, to any person obtaining a copy
 *   of this software and associated documentation files (the "Software"), to deal
 *   in the Software without restriction, including without limitation the rights
 *   to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *   copies of the Software, and to permit persons to whom the Software is
 *   furnished to do so, subject to the following conditions:
 
 *   The above copyright notice and this permission notice shall be included in all
 *   copies or substantial portions of the Software.
 
 *   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *   SOFTWARE.
 */

package component

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
	"github.com/x-research-team/utils/is"
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
			messages := make(JournalMessages, 0)
			if !is.JSON(string(response)) {
				bus.Info <- string(response)
				continue
			}
			err := json.Unmarshal(response, &messages)
			switch {
			case err != nil:
				if err := h.fail(err); err != nil {
					bus.Error <- err
					continue
				}
				bus.Error <- err
				continue
			case messages.IsEmpty():
				if err := h.fail(errors.New("EMPTY_RESPONSE")); err != nil {
					bus.Error <- err
					continue
				}
				continue
			case messages.IsOne():
				m := messages[0]
				if err := h.send(&JournalMessageResponse{
					ID: m.ID,
					Data: m.Data,
				}); err != nil {
					bus.Error <- err
					continue
				}
				continue
			case messages.IsMany():
				response := make(JournalMessagesResponse, 0)
				for _, m := range messages {					
					response = append(response, &JournalMessageResponse{
						ID: m.ID,
						Data: m.Data,
					})
				}
				if err := h.send(response); err != nil {
					bus.Error <- err
					continue
				}
				continue
			default:
				if err := h.fail(errors.New("BAD_REQUEST")); err != nil {
					bus.Error <- err
					continue
				}
			}
		default:
			continue
		}
	}
}

func (h *Hub) fail(e error) error {
	if err := h.send(map[string]string{"error": e.Error()}); err != nil {
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
