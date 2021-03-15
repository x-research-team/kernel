package component

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/x-research-team/bus"
	"github.com/x-research-team/contract"
	"github.com/x-research-team/utils/magic"
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
		component.socket.OnEvent("/", "journal.load", func(s socketio.Conn, msg string) string {
			s.SetContext(msg)
			return "recv " + msg
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
				ctx.JSON(http.StatusInternalServerError, Error(err))
				return
			}
			m := new(KernelMessage)
			if err = json.Unmarshal(buffer, m); err != nil {
				ctx.JSON(http.StatusBadRequest, Error(err))
				return
			}
			message := bus.Message(m.Route, m.Command, string(m.Message))
			component.trunk <- bus.Signal(message)
			ctx.JSON(http.StatusOK, gin.H{"id": message.ID()})
		})
		component.engine.GET("/api", func(ctx *gin.Context) {
			ids := strings.Split(ctx.Query("id"), ",")
			message := bus.Message("storage", "journal", fmt.Sprintf(`{"service":"signal","collection":"messages","filter":{"field":"id","query":"%v"}}`, ids))
			component.trunk <- bus.Signal(message)
			for {
				select {
				case response := <-component.bus:
					messages := make([]JournalMessage, 0)
					bus.Info <- fmt.Sprintf("%v", string(response))
					err := json.Unmarshal(response, &messages)
					switch {
					case err != nil:
						ctx.JSON(http.StatusInternalServerError, Error(err))
						return
					case len(messages) == 0:
						ctx.JSON(http.StatusNotFound, Error(errors.New("empty response")))
						return
					case len(messages) == 1:
						if messages[0].ID != ids[0] {
							continue
						}
						data, err := magic.Jsonify(messages[0].Data)
						if err != nil {
							ctx.JSON(http.StatusInternalServerError, Error(err))
							return
						}
						ctx.JSON(http.StatusOK, JournalMessageResponse{messages[0].ID, data})
						return
					case len(messages) > 1:
						response := make(JournalMessagesResponse, 0)
						for _, m := range messages {
							for _, id := range ids {
								if id != m.ID {
									continue
								}
								data, err := magic.Jsonify(m.Data)
								if err != nil {
									ctx.JSON(http.StatusInternalServerError, Error(err))
									return
								}
								response = append(response, JournalMessageResponse{m.ID, data})
							}
						}
						ctx.JSON(http.StatusOK, response)
						return
					default:
						ctx.JSON(http.StatusBadRequest, Error(errors.New("bad request")))
						return
					}
				default:
					continue
				}
			}
		})
		handler := gin.WrapH(component.socket)
		component.engine.GET("/socket/*any", handler)
		component.engine.POST("/socket/*any", handler)
		component.engine.Handle("WS", "/socket/*any", handler)
		component.engine.Handle("WSS", "/socket/*any", handler)
		c = component
	}
}

func GinMiddleware(allowOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Length, X-CSRF-Token, Token, session, Origin, Host, Connection, Accept-Encoding, Accept-Language, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Request.Header.Del("Origin")

		c.Next()
	}
}

func Error(err error) gin.H {
	bus.Error <- err
	return gin.H{"error": err.Error()}
}

type JournalMessage struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type JournalMessages []JournalMessage

type JournalMessageResponse struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

type JournalMessagesResponse []JournalMessageResponse

type KernelMessage struct {
	Route   string          `json:"route"`
	Command string          `json:"command"`
	Message json.RawMessage `json:"message"`
}
