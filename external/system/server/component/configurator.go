package component

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
)

const JTMP = `{"service":"signal","collection":"messages","filter":{"field":"id","query":"%v"}}`

func configureSocket(component *Component) {
	component.tcpserver = &http.Server{Addr: ":3000", Handler: nil}
	component.socket = newHub(&component.trunk, &component.tcp)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		tcp(component.socket, w, r)
	})
}

func configureHttp(component *Component) {
	component.httpserver = gin.Default()
	component.httpserver.POST("/api", func(ctx *gin.Context) {
		buffer, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, Error(err))
			return
		}
		m := new(KernelMessage)
		if err = json.Unmarshal(buffer, m); err != nil {
			ctx.JSON(http.StatusBadRequest, Error(err))
			return
		}
		message := bus.Message(m.Route, m.Command, string(m.Message))
		go func(m contract.IMessage) { component.trunk <- bus.Signal(m) }(message)
		ctx.JSON(http.StatusOK, gin.H{"id": message.ID()})
		response := bus.Message("storage", "journal-store", fmt.Sprintf(JTMP, message.ID()))
		go func(m contract.IMessage) { component.trunk <- bus.Signal(m) }(response)
	})
	component.httpserver.GET("/api", func(ctx *gin.Context) {
		ids := strings.Split(ctx.Query("id"), ",")
		message := bus.Message("storage", "journal", fmt.Sprintf(JTMP, ids))
		go func(m contract.IMessage) { component.trunk <- bus.Signal(m) }(message)
		for {
			select {
			case <-time.After(time.Microsecond * component.config.Timeout):
				ctx.JSON(http.StatusGatewayTimeout, Error(errors.New("gateway timed out")))
				return
			case response := <-component.bus:
				messages := make(JournalMessages, 0)
				err := json.Unmarshal(response, &messages)
				switch {
				case err != nil:
					ctx.JSON(http.StatusInternalServerError, Error(err))
					return
				case messages.IsEmpty():
					ctx.JSON(http.StatusNotFound, Error(errors.New("NOT_FOUND")))
					return
				case messages.IsOne():
					m := messages[0]
					ctx.JSON(http.StatusOK, &JournalMessageResponse{
						ID:   m.ID,
						Data: m.Data,
					})
					return
				case messages.IsMany():
					response := make(JournalMessagesResponse, 0)
					for _, m := range messages {
						response = append(response, &JournalMessageResponse{
							ID:   m.ID,
							Data: m.Data,
						})
					}
					ctx.JSON(http.StatusOK, response)
					return
				default:
					ctx.JSON(http.StatusBadRequest, Error(errors.New("BAD_REQUEST")))
					return
				}
			default:
				continue
			}
		}
	})
}
func Configure() contract.ComponentModule {
	return func(c contract.IComponent) {
		configureSocket(c.(*Component))
		configureHttp(c.(*Component))
	}
}

func Error(err error) gin.H {
	bus.Error <- err
	return gin.H{"error": err.Error()}
}

type JournalMessage struct {
	ID   string          `json:"id"`
	Data json.RawMessage `json:"data"`
}

type JournalMessages []JournalMessage

func (m JournalMessages) IsEmpty() bool {
	return len(m) == 0
}

func (m JournalMessages) IsOne() bool {
	return len(m) == 1
}

func (m JournalMessages) IsMany() bool {
	return len(m) > 1
}

type JournalMessageResponse struct {
	ID   string          `json:"id"`
	Data json.RawMessage `json:"data"`
}

type JournalMessagesResponse []*JournalMessageResponse

type KernelMessage struct {
	Route   string          `json:"route"`
	Command string          `json:"command"`
	Message json.RawMessage `json:"message"`
}
