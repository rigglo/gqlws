package gqlws

import (
	"context"
	"encoding/json"
	"fmt"
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
	upgrader := websocket.Upgrader{
		Subprotocols: []string{graphQLWS},
		CheckOrigin:  h.conf.CheckOrigin,
	}

	// Upgrading connection to WebSocket
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Couldn't upgrade connection: %s", err)))
		return
	}
	defer socket.Close()

	closers := map[string]func(){}

	for {
		m := new(message)
		err := socket.ReadJSON(m)
		if err != nil {
			return
		}

		switch m.Type {
		case connectionInitMessage:
			send(socket, "", connectionACKMessage, nil)
		case stopMessage:
			closers[m.ID]()
			err = send(socket, m.ID, completeMessage, nil)
			if err != nil {
				send(socket, m.ID, errorMessage, err)
				return
			}
		case connectionTermination:
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

			ctx, cancel := context.WithCancel(r.Context())
			out, err := h.conf.Subscriber(ctx, p.Query, p.OperationName, p.Variables)
			if err != nil {
				send(socket, m.ID, errorMessage, err)
			}
			closers[m.ID] = cancel

			go func() {
				defer socket.Close()
				for v := range out {
					err := send(socket, m.ID, dataMessage, v)
					if err != nil {
						break
					}
				}
				send(socket, m.ID, completeMessage, nil)
				closers[m.ID]()
				delete(closers, m.ID)
			}()
		}
	}
}

func send(socket *websocket.Conn, id string, mt string, payload interface{}) error {
	return socket.WriteJSON(message{
		ID:      id,
		Type:    mt,
		Payload: payload,
	})
}
