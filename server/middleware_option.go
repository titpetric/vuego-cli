package server

import (
	"github.com/titpetric/vuego"
)

// MiddlewareOption configures the middleware behavior.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	loadOptions []vuego.LoadOption
}

// WithLoadOption adds a LoadOption to the middleware's Vue instance.
func WithLoadOption(opt ...vuego.LoadOption) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.loadOptions = append(cfg.loadOptions, opt...)
	}
}
