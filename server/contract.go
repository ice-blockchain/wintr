// SPDX-License-Identifier: BUSL-1.1

package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Public API.

var (
	ErrSomethingWentWrong = errors.New("oops, something went wrong")
	ErrNotAuthenticated   = errors.New("not authenticated")
)

type (
	Server interface {
		// ListenAndServe starts everything and blocks indefinitely.
		ListenAndServe(ctx context.Context, cancel context.CancelFunc)
	}
	// AuthenticatedUser is the payload structure extracted from the Authorization header, after a successful authentication.
	AuthenticatedUser struct {
		// ID is the token`s issuer.
		ID string `json:"id"`
	}
	// State is the actual custom behaviour that has to be implemented by users of this package to customize their http server`s lifecycle.
	State interface {
		Init(context.Context, context.CancelFunc)
		Close(context.Context) error
		RegisterRoutes(*gin.Engine)
		CheckHealth(context.Context, *RequestCheckHealth) Response
	}
	RequestCheckHealth struct {
		ClientIP net.IP `json:"clientIP" swaggerignore:"true"`
	}
	// ParsedRequest has to be implemented by every endpoint`s request struct.
	ParsedRequest interface {
		// Validate is responsible for validating the struct after binding it properly and returning http.StatusBadRequest.
		Validate() *Response
		// Bindings are responsible for parsing structs (via tags or custom impl) and returning http.StatusUnprocessableEntity.
		Bindings(*gin.Context) []func(interface{}) error
	}
	// ClientIPSetGetter is a marker interface that should be implemented by request structs that need access to the Client`s actual IP.
	ClientIPSetGetter interface {
		SetClientIP(net.IP)
		GetClientIP() net.IP
	}
	// AuthenticatedUserSetGetter is a marker interface that should be implemented by request structs that need access to AuthenticatedUser.
	AuthenticatedUserSetGetter interface {
		SetAuthenticatedUser(AuthenticatedUser)
		GetAuthenticatedUser() AuthenticatedUser
	}
	// Response is the structure returned and handled by the router.
	//	Data can be any JSON serializable struct, error or ErrorResponse.
	Response struct {
		Data interface{}
		Code int
	}
	// ErrorResponse is the struct that is eventually serialized as a negative response back to the user.
	ErrorResponse struct {
		error `json:"-" swaggerignore:"true"`
		Data  map[string]interface{} `json:"data,omitempty"`
		Error string                 `json:"error" example:"something is missing"`
		Code  string                 `json:"code,omitempty" example:"SOMETHING_NOT_FOUND"`
	}
)

// Private API.

const authenticatedUserGinCtxKey = "authenticatedUser"

var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	development bool
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	cfg config
)

type (
	// | srv is the internal representation of everything needed to bootstrap the http server.
	srv struct {
		State
		server      *http.Server
		router      *gin.Engine
		swaggerRoot string
	}
	config struct {
		HTTPServer struct {
			CertPath string `yaml:"certPath"`
			KeyPath  string `yaml:"keyPath"`
			Port     uint16 `yaml:"port"`
		} `yaml:"httpServer"`
		DefaultEndpointTimeout time.Duration `yaml:"defaultEndpointTimeout"`
	}
)
