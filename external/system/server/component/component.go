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
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"

	"github.com/google/uuid"
)

const (
	name  = "Server"
	route = "server"
)

type config struct {
	Timeout time.Duration `json:"timeout"`
}

// Component
type Component struct {
	bus chan []byte
	tcp chan []byte

	config *config

	components map[string]contract.IComponent
	trunk      contract.ISignalBus
	route      string
	uuid       string
	fails      []error

	httpserver *gin.Engine
	tcpserver  *http.Server
	socket     *Hub
}

// New Создать экземпляр компонента сервиса биллинга
func New(opts ...contract.ComponentModule) contract.KernelModule {
	component := &Component{
		bus:        make(chan []byte),
		tcp:        make(chan []byte),
		components: make(map[string]contract.IComponent),
		route:      route,
		trunk:      make(contract.ISignalBus),
		config:     new(config),
	}
	for _, o := range opts {
		o(component)
	}
	if len(component.fails) > 0 {
		for _, err := range component.fails {
			bus.Error <- fmt.Errorf("[%s] %v", name, err)
		}
		return func(service contract.IService) {
		}
	}
	bus.Add(component.trunk)
	bus.Info <- fmt.Sprintf("[%v] Initialized", name)
	return func(c contract.IService) {
		c.AddComponent(component)
		bus.Info <- fmt.Sprintf("[%v] attached to Billing Service", name)
	}
}

func (component *Component) AddComponent(c contract.IComponent) {
	component.components[c.Name()] = c
}

// Send Отправить сигнал в ядро
func (component *Component) Send(message contract.IMessage) {
	component.trunk.Send(bus.Signal(message))
}

// AddPlugin Добавить плагин на горячем ходу
func (component *Component) AddPlugin(p, name string) error {
	return nil
}

// RemovePlugin Удалить плагин на горячем ходу
func (component *Component) RemovePlugin(name string) error {
	return nil
}

// Configure Конфигурация компонета платежной системы
func (component *Component) Configure() error {
	bus.Info <- fmt.Sprintf("[%v] is configured", name)
	return nil
}

// Run Запуск компонента платежной системы
func (component *Component) Run() error {
	bus.Info <- fmt.Sprintf("[%v] component started", name)
	component.uuid = uuid.New().String()
	go component.socket.run()
	go func () {
		if err := component.tcpserver.ListenAndServe(); err != nil {
			bus.Error <- err
			return
		}
	}()
	return component.httpserver.Run(":43001")
}

func (component *Component) Route() string { return component.route }

func (component *Component) Write(message contract.IMessage) error {
	if message.Route() != component.Route() {
		return nil
	}
	data := message.Data()
	go func (m string) { component.bus <- []byte(m) }(data)
	go func (m string) { component.tcp <- []byte(m) }(data)
	return nil
}

func (component *Component) Read() string {
	return ""
}

func (component *Component) Pid() string {
	return component.uuid
}

func (component *Component) Name() string {
	return name
}

func (component *Component) Up(graceful bool) error {
	return nil
}

func (component *Component) Down(graceful bool) error {
	return nil
}

func (component *Component) Sleep(time.Duration) error {
	return nil
}

func (component *Component) Restart(graceful bool) error {
	return nil
}

func (component *Component) Pause() error {
	return nil
}

func (component *Component) Cron(rule string) error {
	return nil
}

func (component *Component) Stop() error {
	return nil
}

func (component *Component) Kill() error {
	return nil
}

func (component *Component) Sync(with string) error {
	return nil
}

func (component *Component) Backup(to string) error {
	return nil
}
