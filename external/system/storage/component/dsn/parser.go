package dsn

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/kernel/external/system/storage/component/dialect"
	"github.com/x-research-team/utils/file"
)

var m sync.Mutex

type TDataBaseConfig struct {
	Name     string `json:"name"`
	Dialect  string `json:"dialect"`
	Database string `json:"database"`
}

func (c TDataBaseConfig) GetDialect() string {
	return c.Dialect
}

func (c TDataBaseConfig) GetDSN() string {
	return ""
}

func (c TDataBaseConfig) GetUser() string {
	return ""
}

func (c TDataBaseConfig) GetPassword() string {
	return ""
}

type IDataBaseConfig interface {
	GetDialect() string
	GetDSN() string
	GetUser() string
	GetPassword() string
}

type TMySQLConfig struct {
	TDataBaseConfig
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     uint   `json:"port"`
}

func (c TMySQLConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.User, c.Password, c.Host, c.Port, c.Database)
}

type TSQLiteConfig struct {
	TDataBaseConfig
}

func (c TSQLiteConfig) GetDSN() string {
	return fmt.Sprintf("file:%s?cache=shared&mode=rwc", c.TDataBaseConfig.Database)
}

type TPostgresConfig struct {
	TDataBaseConfig
}

type TMongoConfig struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	TDataBaseConfig
}

func (c TMongoConfig) GetDialect() string {
	return "mongodb"
}

func (c TMongoConfig) GetDSN() string {
	return fmt.Sprintf("%s://%s:%d", c.GetDialect(), c.Host, c.Port)
}

func (c TMongoConfig) GetUser() string {
	return c.User
}

func (c TMongoConfig) GetPassword() string {
	return c.Password
}

func Parse() map[string]IDataBaseConfig {
	v := make(map[string]IDataBaseConfig)
	dbconfigs, err := filepath.Glob("**/*.dbconfig")
	if err != nil {
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(len(dbconfigs))
	for _, dbconfig := range dbconfigs {
		go func(wg *sync.WaitGroup, dbconfig string) {
			defer wg.Done()
			c := new(TDataBaseConfig)
			if err := file.Read(".", dbconfig, c); err != nil {
				bus.Error <- err
				return
			}
			if c.Name == "" {
				bus.Error <- fmt.Errorf("name of connection can not be empty")
				return
			}
			var s IDataBaseConfig
			switch c.GetDialect() {
			case dialect.MySQL:
				s = new(TMySQLConfig)
			case dialect.SQLite:
				s = new(TSQLiteConfig)
			case dialect.Postgres:
				s = new(TPostgresConfig)
			case dialect.Mongo:
				s = new(TMongoConfig)
			}
			if err := file.Read(".", dbconfig, s); err != nil {
				bus.Error <- err
				return
			}
			m.Lock()
			v[c.Name] = s
			m.Unlock()
		}(&wg, dbconfig)
	}
	wg.Wait()
	return v
}
