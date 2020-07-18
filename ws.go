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
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type errorPayload struct {
	Message string `json:"message"`
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
	ctx := r.Context()

	for {
		m := new(message)
		err := socket.ReadJSON(m)
		if err != nil {
			return
		}

		switch m.Type {
		case connectionInitMessage:
			p := map[string]interface{}{}
			err = json.Unmarshal(m.Payload, &p)
			if err != nil {
				send(socket, m.ID, errorMessage, errorPayload{err.Error()})
				return
			}
			if ctx, err = h.conf.OnConnect(ctx, p); err != nil {
				send(socket, m.ID, errorMessage, errorPayload{err.Error()})
				return
			}
			send(socket, "", connectionACKMessage, nil)
		case stopMessage:
			closers[m.ID]()
			err = send(socket, m.ID, completeMessage, nil)
			if err != nil {
				send(socket, m.ID, errorMessage, errorPayload{err.Error()})
				return
			}
		case connectionTermination:
			socket.Close()
			return
		case startMessage:
			p := new(opPayload)
			err = json.Unmarshal(m.Payload, &p)
			if err != nil {
				send(socket, m.ID, errorMessage, errorPayload{err.Error()})
				return
			}

			ctx, cancel := context.WithCancel(ctx)
			out, err := h.conf.Subscriber(ctx, p.Query, p.OperationName, p.Variables)
			if err != nil {
				send(socket, m.ID, errorMessage, errorPayload{err.Error()})
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
				cancel()
				delete(closers, m.ID)
			}()
		}
	}
}

type sentMessage struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func send(socket *websocket.Conn, id string, mt string, payload interface{}) error {
	return socket.WriteJSON(sentMessage{
		ID:      id,
		Type:    mt,
		Payload: payload,
	})
}
