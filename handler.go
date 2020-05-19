package gqlws

import (
	"context"
	"net/http"
)

// Subscriber lets you use any GraphQL implementation that
type Subscriber func(ctx context.Context, query string, operationName string, variables map[string]interface{}) (<-chan interface{}, error)

type Config struct {
	CheckOrigin func(r *http.Request) bool
	UpgradeRule func(r *http.Request) bool
	Subscriber  Subscriber
}

func New(c Config, next http.Handler) http.Handler {
	h := &handler{
		conf: Config{
			CheckOrigin: func(r *http.Request) bool { return true },
			UpgradeRule: func(r *http.Request) bool { return true },
		},
		next: next,
	}
	if c.CheckOrigin != nil {
		h.conf.CheckOrigin = c.CheckOrigin
	}
	if c.UpgradeRule != nil {
		h.conf.UpgradeRule = c.UpgradeRule
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
