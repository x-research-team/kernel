package component

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
)

func Configure() contract.ComponentModule {
	return func(c contract.IComponent) {
		component := c.(*Component)
		component.engine = gin.Default()
		var err error
		if component.socket, err = socketio.NewServer(nil); err != nil {
			bus.Error <- err
			return
		}
		component.socket.OnConnect("/", func(s socketio.Conn) error {
			s.SetContext("")
			fmt.Println("connected:", s.ID())
			return nil
		})
		component.socket.OnError("/", func(s socketio.Conn, e error) {
			fmt.Println("meet error:", e)
		})
		component.socket.OnDisconnect("/", func(s socketio.Conn, msg string) {
			fmt.Println("closed", msg)
		})
		component.engine.POST("/api", func(ctx *gin.Context) {
			buffer, err := ioutil.ReadAll(ctx.Request.Body)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"Error": err})
				return
			}
			m := new(KernelMessage)
			if err = json.Unmarshal(buffer, m); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
				return
			}
			message := bus.Message(m.Route, m.Command, m.Message)
			component.trunk <- bus.Signal(message)
			ctx.JSON(http.StatusOK, gin.H{"id": message.ID()})
		})
		component.engine.GET("/api", func(ctx *gin.Context) {
			id := strings.Split(ctx.Query("id"), ",")
			message := bus.Message("storage", "journal", fmt.Sprintf(`{"service":"signal","collection":"messages","filter":{"field":"id","query":"%v"}}`, id))
			component.trunk <- bus.Signal(message)
			for {
				select {
				case response := <-component.bus:
					m := make([]JournalMessage, 0)
					bus.Info <- fmt.Sprintf("%v", string(response))
					err := json.Unmarshal(response, &m)
					switch {
					case err != nil:
						ctx.JSON(http.StatusInternalServerError, gin.H{"Error": err.Error()})
						return
					case len(m) == 0:
						ctx.JSON(http.StatusNotFound, m)
						return
					case len(m) == 1:
						if m[0].ID != id[0] {
							continue
						}
						ctx.JSON(http.StatusOK, m[0])
						return
					case len(m) > 1:
						n := make([]JournalMessage, 0)
						for _, i := range m {
							for _, j := range id {
								if j != i.ID {
									continue
								}
								n = append(n, i)
							}
						}
						ctx.JSON(http.StatusOK, n)
						return
					default:
						ctx.JSON(http.StatusBadRequest, nil)
						return
					}
				default:
					continue
				}
			}
		})
		component.engine.GET("/socket/*any", gin.WrapH(component.socket))
		component.engine.POST("/socket/*any", gin.WrapH(component.socket))
		c = component
	}

}

type JournalMessage struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type KernelMessage struct{ Route, Command, Message string }
