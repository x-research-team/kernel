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

package kernel

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
	"github.com/x-research-team/implant"
	"github.com/x-research-team/vm"
)

// Kernel Сервис биллинга
type Kernel struct {
	components map[string]contract.IComponent // Набор компонентов ядра

	uuid string
}

// New Создать экземпляр сервиса биллинга
func New(opts ...contract.KernelModule) contract.IService {
	b := &Kernel{components: make(map[string]contract.IComponent)}
	for _, o := range opts {
		o(b)
	}
	bus.Info <- "[Kernel] Service initialized"
	vm.RegisterFunctions("signal", map[string]interface{}{
		"New":     bus.Signal,
		"Message": bus.Message,
	})
	return b
}

// AddPlugin Добавить плагин на горячем ходу
func (kernel *Kernel) AddPlugin(p, name string) error {
	implant.Init(p)
	for _, o := range implant.Modules() {
		o(kernel)
	}
	kernel.run(kernel.components[name])
	return nil
}

// RemovePlugin Удалить плагин на горячем ходу
func (kernel *Kernel) RemovePlugin(name string) error {
	delete(kernel.components, name)
	return nil
}

// Run Запуск сервиса биллинга
func (kernel *Kernel) Run() error {
	kernel.uuid = uuid.New().String()
	for n := range kernel.components {
		go kernel.run(kernel.components[n])
	}
	bus.Info <- "[Kernel] Service started"
	for true {
		for signal := range bus.Signals.Generate() {
			go kernel.signal(signal.Message())
		}
	}

	return nil
}

func (kernel *Kernel) signal(m contract.IMessage) {
	route := m.Route()
	if route == "" {
		bus.Error <- fmt.Errorf("route %v is not a found", route)
		return
	}
	for k := range kernel.components {
		go kernel.handle(kernel.components[k], m)
	}
}

// run Запуск компонетов ядра
func (kernel *Kernel) run(p contract.IComponent) {
	if err := p.Configure(); err != nil {
		bus.Error <- err
	}
	defer func() {
		if r := recover(); r != nil {
			bus.Error <- fmt.Errorf("[%s] failed to start", p.Name())
		}
	}()
	if err := p.Run(); err != nil {
		bus.Error <- err
	}
	select {}
}

// handle Обработка данных конкретным компонентом ядра
func (kernel *Kernel) handle(c contract.IComponent, message contract.IMessage) {
	if err := c.Write(message); err != nil {
		bus.Error <- err
	}
}

func (kernel *Kernel) AddComponent(c contract.IComponent) {
	kernel.components[c.Name()] = c
}

func (kernel *Kernel) Pid() string {
	return kernel.uuid
}

func (kernel *Kernel) Name() string {
	return "Kernel"
}

func (kernel *Kernel) Up(graceful bool) error {
	return nil
}

func (kernel *Kernel) Down(graceful bool) error {
	return nil
}

func (kernel *Kernel) Sleep(time.Duration) error {
	return nil
}

func (kernel *Kernel) Restart(graceful bool) error {
	return nil
}

func (kernel *Kernel) Pause() error {
	return nil
}

func (kernel *Kernel) Cron(rule string) error {
	return nil
}

func (kernel *Kernel) Stop() error {
	return nil
}

func (kernel *Kernel) Kill() error {
	return nil
}

func (kernel *Kernel) Sync(with string) error {
	return nil
}

func (kernel *Kernel) Backup(to string) error {
	return nil
}
