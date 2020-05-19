package gqlws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	connectionInitMessage      = "connection_init"
	connectionACKMessage       = "connection_ack"
	connectionErrorMessage     = "connection_error"
	connectionKeepAliveMessage = "ka"
	connectionTermination      = "connection_terminate"
	startMessage               = "start"
	dataMessage                = "data"
	errorMessage               = "error"
	completeMessage            = "complete"
	stopMessage                = "stop"

	graphQLWS = "graphql-ws"
)

type opPayload struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

type message struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func shouldUpgrade(r *http.Request) bool {
	for _, s := range websocket.Subprotocols(r) {
		if s == graphQLWS {
			return true
		}
	}
	return false
}

func (h *handler) subscription(w http.ResponseWriter, r *http.Request) {
	log.Println("new connection")
	upgrader := websocket.Upgrader{
		Subprotocols: []string{graphQLWS},
		CheckOrigin:  h.conf.CheckOrigin,
	}

	// Upgrading connection to WebSocket
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Error upgrading connection: %s", err)))
		return
	}
	defer socket.Close()
	defer log.Println("end connection")

	closers := map[string]func(){}

	for {
		_, msgBytes, err := socket.ReadMessage()
		if err != nil {
			log.Println("err:", string(msgBytes))
		}
		m := new(message)
		err = json.Unmarshal(msgBytes, m)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(string(msgBytes))
		switch m.Type {
		case connectionInitMessage:
			err = socket.WriteJSON(message{
				Type: connectionACKMessage,
			})
			if err != nil {
				log.Println(err)
				return
			}
		case stopMessage, connectionTermination:
			err = send(socket, m.ID, completeMessage, nil)
			if err != nil {
				log.Println(err)
				return
			}
			socket.Close()
			return
		case startMessage:
			p := new(opPayload)
			pl, err := json.Marshal(m.Payload)
			if err != nil {
				send(socket, m.ID, errorMessage, err)
				return
			}
			err = json.Unmarshal(pl, &p)
			if err != nil {
				send(socket, m.ID, errorMessage, err)
				return
			}
			if err != nil {
				send(socket, m.ID, errorMessage, nil)
				return
			}
			ctx, cancel := context.WithCancel(r.Context())
			out, err := h.conf.Subscriber(ctx, p.Query, p.OperationName, p.Variables)
			if err != nil {
				log.Printf("error from exec.Subscribe(): %s", err)
				send(socket, m.ID, connectionErrorMessage, nil)
			}
			closers[m.ID] = cancel

			go func() {
				for v := range out {
					send(socket, m.ID, dataMessage, v)
				}
				send(socket, m.ID, completeMessage, nil)
				closers[m.ID]()
				delete(closers, m.ID)
			}()
		}
	}
}

func send(socket *websocket.Conn, id string, mt string, payload interface{}) error {
	log.Printf("sent: %s, %s, %+v", id, mt, payload)
	return socket.WriteJSON(message{
		ID:      id,
		Type:    mt,
		Payload: payload,
	})
}
