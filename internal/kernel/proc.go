package kernel

import (
	"fmt"
	"runtime"
	"time"

	"github.com/x-research-team/vm"

	"github.com/google/uuid"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
	"github.com/x-research-team/implant"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

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
	bus.Debug <- fmt.Sprintf("%v: %v => %s", route, m.Command(), m.Data())
	if route != "" {
		for k := range kernel.components {
			go kernel.handle(kernel.components[k], m)
		}
	} else {
		bus.Error <- fmt.Errorf("Route %v is not a found", route)
	}
}

// run Запуск компонетов ядра
func (kernel *Kernel) run(p contract.IComponent) {
	if err := p.Configure(); err != nil {
		bus.Error <- err
	}
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
