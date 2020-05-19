package gqlws

import (
	"context"
	"errors"
	"net/http"
)

// Subscriber lets you use any GraphQL implementation you want
type Subscriber func(ctx context.Context, query string, operationName string, variables map[string]interface{}) (<-chan interface{}, error)

// Config for the GraphQL subscriptions over WebSocket
type Config struct {
	// CheckOrigin can check the origin of a request, by default it allows everything
	CheckOrigin func(*http.Request) bool
	// UpgradeRule lets you define your own rule to limit which request you want to upgrade to subscriptions and which you do not
	UpgradeRule func(*http.Request) bool
	// Subscriber is a function from the GraphQL implementation you use to provide a result channer for gqlws
	Subscriber Subscriber
}

// New returns a new handler with the given config
func New(c Config, next http.Handler) http.Handler {
	h := &handler{
		conf: Config{
			CheckOrigin: func(r *http.Request) bool { return true },
			UpgradeRule: func(r *http.Request) bool { return true },
			Subscriber: func(ctx context.Context, query string, operationName string, variables map[string]interface{}) (<-chan interface{}, error) {
				return nil, errors.New("no subscriber function provided")
			},
		},
		next: next,
	}
	if c.CheckOrigin != nil {
		h.conf.CheckOrigin = c.CheckOrigin
	}
	if c.UpgradeRule != nil {
		h.conf.UpgradeRule = c.UpgradeRule
	}
	if c.Subscriber != nil {
		h.conf.Subscriber = c.Subscriber
	}
	return h
}

type handler struct {
	conf Config
	next http.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if shouldUpgrade(r) && h.conf.UpgradeRule(r) {
		h.subscription(w, r)
		return
	}
	h.next.ServeHTTP(w, r)
}
