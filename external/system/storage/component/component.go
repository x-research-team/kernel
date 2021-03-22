package component

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bdwilliams/go-jsonify/jsonify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	. "github.com/Masterminds/squirrel"

	"github.com/google/uuid"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
	"github.com/x-research-team/utils/magic"
)

const (
	name  = "Storage"
	route = "storage"
)

// Component
type Component struct {
	bus chan []byte

	components map[string]contract.IComponent
	trunk      contract.ISignalBus
	route      string
	uuid       string

	client  map[string]*sql.DB
	journal map[string]*mongo.Client
	fails   []error
}

// New Создать экземпляр компонента сервиса биллинга
func New(opts ...contract.ComponentModule) contract.KernelModule {
	component := &Component{
		bus:        make(chan []byte),
		components: make(map[string]contract.IComponent),
		route:      route,
		trunk:      make(contract.ISignalBus),
		client:     make(map[string]*sql.DB),
		journal:    make(map[string]*mongo.Client),
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
	c := component.journal["signal"]
	if c == nil {
		return errors.New("connection (signal) not found")
	}
	signal := c.Database("signal")
	signal.Collection("messages")
	return nil
}

// Run Запуск компонента платежной системы
func (component *Component) Run() error {
	bus.Info <- fmt.Sprintf("[%v] component started", name)

	component.uuid = uuid.New().String()

	var RequestSyncronizer sync.Map

	for {
		select {
		case data := <-component.bus:
			fmt.Printf("%s\n", data)
			m := new(KernelMessage)
			if err := json.Unmarshal(data, &m); err != nil {
				bus.Error <- err
				continue
			}
			command := new(TCommand)
			if err := json.Unmarshal(m.Data, &command); err != nil {
				bus.Error <- err
				continue
			}
			var (
				result []map[string]interface{}
				err    error
			)
			syncID := "stored" + ":" + m.ID.String()
			switch m.Command {
			case "journal-store":
				go func(m KernelMessage) {
					command := new(TCommand)
					if err := json.Unmarshal(m.Data, &command); err != nil {
						bus.Error <- err
						return
					}
					releasedSyncID := "stored" + ":" + command.Filter.Query
					fmt.Println("RequestSyncronizer.Load", releasedSyncID, "loaded")
					if ready, exists := RequestSyncronizer.Load(releasedSyncID); exists {
						for {
							switch {
							case ready:
								if result, err = component.load(command); err != nil {
									bus.Error <- err
									if err := component.signal(m.ID.String(), nil, err); err != nil {
										bus.Error <- err
										continue
									}
									continue
								}
								RequestSyncronizer.Delete(releasedSyncID)
								fmt.Println("RequestSyncronizer.Delete", releasedSyncID, "deleted")
								buffer, err := json.Marshal(result)
								if err != nil {
									bus.Error <- err
									return
								}
								fmt.Println(string(buffer))
								component.trunk <- bus.Signal(bus.Message("server", "response", string(buffer)))
								return
							default:
								time.Sleep(time.Microsecond * 500)
								continue
							}
						}
					}
				}(*m)
				continue
			case "journal":
				if result, err = component.load(command); err != nil {
					bus.Error <- err
					if err := component.signal(m.ID.String(), nil, err); err != nil {
						bus.Error <- err
						continue
					}
					continue
				}
				RequestSyncronizer.Delete(syncID)
				buffer, err := json.Marshal(result)
				if err != nil {
					bus.Error <- err
					continue
				}
				component.trunk <- bus.Signal(bus.Message("server", "response", string(buffer)))
				continue
			case "store":
				RequestSyncronizer.Store(syncID, false)
				fmt.Println("RequestSyncronizer.Store", syncID, false)
				if result, err = component.handle(command); err != nil {
					bus.Error <- err
					if err := component.signal(m.ID.String(), nil, err); err != nil {
						bus.Error <- err
						continue
					}
					continue
				}
			default:
				err := fmt.Errorf("unknown command (%v)", m.Command)
				bus.Error <- err
				if err := component.signal(m.ID.String(), nil, err); err != nil {
					bus.Error <- err
					continue
				}
				continue
			}
			if err := component.signal(m.ID.String(), result, nil); err != nil {
				bus.Error <- err
				continue
			}
			RequestSyncronizer.Store(syncID, true)
			fmt.Println("RequestSyncronizer.Store", syncID, true)
		default:
			continue
		}
	}
}

func (component *Component) signal(id string, buffer []map[string]interface{}, e error) error {
	c := component.journal["signal"]
	if c == nil {
		return errors.New("connection (signal) not found")
	}
	var data interface{}
	if e != nil {
		data = map[string]interface{}{"error": e.Error()}
	} else {
		data = buffer
	}
	signal := c.Database("signal")
	messages := signal.Collection("messages")
	ctx := context.Background()
	if _, err := messages.InsertOne(ctx, bson.D{{Key: "id", Value: id}, {Key: "data", Value: data}}); err != nil {
		return err
	}
	return nil
}

func (component *Component) storage(tx *sql.Tx) func(Sqlizer) ([]string, error) {
	return func(s Sqlizer) ([]string, error) {
		var empty []string
		query, params, err := s.ToSql()
		if err != nil {
			return empty, err
		}
		stmt, err := tx.Prepare(query)
		if err != nil {
			return empty, err
		}
		switch {
		case strings.HasPrefix(query, "SELECT"):
			rows, err := stmt.Query(params...)
			if err != nil {
				if err = tx.Rollback(); err != nil {
					return empty, err
				}
				return empty, err
			}
			defer func(rows *sql.Rows) {
				if err = rows.Close(); err != nil {
					bus.Error <- err
				}
			}(rows)
			if err = tx.Commit(); err != nil {
				return empty, err
			}
			return jsonify.Jsonify(rows), nil
		case strings.HasPrefix(query, "INSERT"),
			strings.HasPrefix(query, "UPDATE"),
			strings.HasPrefix(query, "DELETE"):
			bus.Info <- fmt.Sprintf("%v", params)
			if _, err := stmt.Exec(params...); err != nil {
				if err := tx.Rollback(); err != nil {
					return empty, err
				}
				return empty, err
			}
		default:
			if err := tx.Rollback(); err != nil {
				return empty, err
			}
			return empty, fmt.Errorf("sql => %s", query)
		}
		if err := tx.Commit(); err != nil {
			bus.Error <- err
			return empty, err
		}
		bus.Info <- "Commit transaction"
		return empty, err
	}
}

func (component *Component) Route() string { return component.route }

type KernelMessage struct {
	ID      uuid.UUID
	Command string
	Data    []byte
}

func (component *Component) Write(message contract.IMessage) error {
	if message.Route() != component.Route() {
		return nil
	}
	bus.Debug <- fmt.Sprintf("%#v", message)
	buffer, err := json.Marshal(&KernelMessage{
		ID:      message.ID(),
		Command: message.Command(),
		Data:    []byte(message.Data()),
	})
	if err != nil {
		return err
	}
	component.bus <- buffer
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

func (component *Component) load(command *TCommand) ([]map[string]interface{}, error) {
	if command.Service == "" {
		return nil, fmt.Errorf("unknown service")
	}
	c := component.journal[command.Service]
	if c == nil {
		return nil, errors.New("connection not found")
	}
	db := c.Database(command.Service)
	collection := db.Collection(command.Collection)
	ctx := context.Background()
	query := strings.ReplaceAll(command.Filter.Query, "[", "")
	query = strings.ReplaceAll(query, "]", "")
	cursor, err := collection.Find(ctx, bson.M{command.Filter.Field: bson.M{"$in": strings.Split(query, " ")}})
	if err != nil {
		return nil, err
	}
	results := make([]map[string]interface{}, 0)
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (component *Component) handle(command *TCommand) ([]map[string]interface{}, error) {
	if command.Service == "" {
		return nil, fmt.Errorf("unknown service")
	}
	if command.SQL == "" {
		return nil, fmt.Errorf("missing sql raw")
	}
	c := component.client[command.Service]
	if c == nil {
		return nil, errors.New("connection not found")
	}
	tx, err := c.Begin()
	if err != nil {
		return nil, err
	}
	stmt, err := tx.Prepare(command.SQL)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}
	if strings.HasPrefix(strings.ToLower(command.SQL), "select") {
		rows, err := stmt.Query()
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return nil, err
			}
			return nil, err
		}
		defer func(rows *sql.Rows) {
			if err = rows.Close(); err != nil {
				bus.Error <- err
			}
		}(rows)
		returns := make([]map[string]interface{}, 0)
		results := jsonify.Jsonify(rows)
		for _, result := range results {
			if result == "," {
				continue
			}
			m, err := magic.Jsonify(result)
			if err != nil {
				if err := tx.Rollback(); err != nil {
					return nil, err
				}
				return nil, err
			}
			returns = append(returns, m)
		}

		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return returns, nil
	}
	if _, err := stmt.Exec(); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return nil, nil
}
