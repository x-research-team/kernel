package component

import (
	"context"
	"errors"
	"time"

	"github.com/x-research-team/kernel/external/system/storage/component/dialect"
	"github.com/x-research-team/kernel/external/system/storage/component/dsn"

	"entgo.io/ent/dialect/sql"
	"github.com/x-research-team/contract"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectTo(dsn map[string]dsn.IDataBaseConfig) contract.ComponentModule {
	return func(component contract.IComponent) {
		c := component.(*Component)
		for k, v := range dsn {
			d := v.GetDialect()
			switch d {
			case dialect.Mongo + "db":
				client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(v.GetDSN()).SetAuth(options.Credential{
					Username: v.GetUser(),
					Password: v.GetPassword(),
				}))
				if err != nil {
					c.fails = append(c.fails, err)
					return
				}
				c.journal[k] = client
			case dialect.MySQL,
				dialect.SQLite,
				dialect.Postgres,
				dialect.Gremlin:
				drv, err := sql.Open(d, v.GetDSN())
				if err != nil {
					c.fails = append(c.fails, err)
					return
				}
				db := drv.DB()
				db.SetMaxIdleConns(10)
				db.SetMaxOpenConns(100)
				db.SetConnMaxLifetime(time.Hour)
				if err = db.Ping(); err != nil {
					c.fails = append(c.fails, err)
					return
				}
				c.client[k] = db
			default:
				c.fails = append(c.fails, errors.New("unsupported dialect"))
				return
			}
		}
		component = c
	}
}
