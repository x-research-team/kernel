package component

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
)

func Configure() contract.ComponentModule {
	return func(c contract.IComponent) {
		component := c.(*Component)
		component.engine = gin.Default()
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
			ctx.JSON(http.StatusOK, gin.H{"ID": message.ID()})
		})
		component.engine.GET("/api", func(ctx *gin.Context) {
			id := ctx.Query("id")
			message := bus.Message("storage", "journal", fmt.Sprintf(`{"Service":"signal","Collection":"messages","Filter":{"id":"%s"}}`, id))
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
						if m[0].ID != id {
							continue
						}
						ctx.JSON(http.StatusOK, m[0])
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
		c = component
	}

}

type JournalMessage struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type KernelMessage struct{ Route, Command, Message string }
