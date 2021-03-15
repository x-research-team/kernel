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
				ctx.JSON(http.StatusInternalServerError, gin.H{"Error": err})
				return
			}
			m := new(KernelMessage)
			if err = json.Unmarshal(buffer, m); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
				return
			}
			message := bus.Message(m.Route, m.Command, string(m.Message))
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
						bus.Error <- err
						ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					case len(m) == 0:
						bus.Error <- errors.New("empty response")
						ctx.JSON(http.StatusNotFound, m)
						return
					case len(m) == 1:
						if m[0].ID != id[0] {
							continue
						}
						data, err := jsonify(m[0].Data)
						if err != nil {
							bus.Error <- err
							ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
							return
						}
						ctx.JSON(http.StatusOK,  JournalMessageResponse{m[0].ID, data})
						return
					case len(m) > 1:
						n := make([]JournalMessageResponse, 0)
						for _, i := range m {
							for _, j := range id {
								if j != i.ID {
									continue
								}
								data, err := jsonify(i.Data)
								if err != nil {
									bus.Error <- err
									ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
									return
								}
								n = append(n, JournalMessageResponse{i.ID, data})
							}
						}
						ctx.JSON(http.StatusOK, n)
						return
					default:
						bus.Error <- errors.New("bad request")
						ctx.JSON(http.StatusBadRequest, nil)
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

func jsonify(s string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		if strings.Contains(err.Error(), "array") {
			array := make([]map[string]interface{}, 0)
			if err := json.Unmarshal([]byte(s), &array); err != nil {
				return nil, err
			}
			for i := range array {
				for k, v := range array[i] {
					if k == "data" {
						switch t := v.(type) {
						case string:
							array[i][k], err = jsonify(t)
							return array[i], err
						default:
							continue
						}
					}
				}
			}
		} else {
			return nil, err
		}
	}
	for k, v := range result {
		if k == "data" {
			switch t := v.(type) {
			case string:
				var err error
				result[k], err = jsonify(t)
				return result, err
			default:
				continue
			}
		}
	}
	return result, nil
}

type JournalMessage struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type JournalMessageResponse struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

type KernelMessage struct {
	Route   string          `json:"route"`
	Command string          `json:"command"`
	Message json.RawMessage `json:"message"`
}
