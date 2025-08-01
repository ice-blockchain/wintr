// SPDX-License-Identifier: ice License 1.0

package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ice-blockchain/wintr/auth"
)

// Public API.

type (
	Router = gin.Engine
	Server interface {
		// ListenAndServe starts everything and blocks indefinitely.
		ListenAndServe(ctx context.Context, cancel context.CancelFunc)
	}
	// AuthenticatedUser is the payload structure extracted from the Authorization header, after a successful authentication.
	AuthenticatedUser struct {
		auth.Token
		Language string
	}
	// State is the actual custom behaviour that has to be implemented by users of this package to customize their http server`s lifecycle.
	State interface {
		Init(ctx context.Context, cancel context.CancelFunc)
		Close(ctx context.Context) error
		RegisterRoutes(r *Router)
		CheckHealth(ctx context.Context) error
	}
	Request[REQ any, RESP any] struct {
		Data                         *REQ                        `json:"data,omitempty"`
		ginCtx                       *gin.Context                //nolint:structcheck // Wrong.
		AuthenticatedUser            AuthenticatedUser           `json:"authenticatedUser,omitempty"`
		ClientIP                     net.IP                      `json:"clientIp,omitempty"`
		bindings                     map[requestBinding]struct{} //nolint:structcheck // Wrong.
		requiredFields               []string                    //nolint:structcheck // Wrong.
		allowUnauthorized            bool                        //nolint:structcheck // Wrong.
		allowForbiddenGet            bool                        //nolint:structcheck // Wrong.
		allowForbiddenWriteOperation bool                        //nolint:structcheck // Wrong.
	}
	Response[RESP any] struct {
		Data    *RESP
		Headers map[string]string
		Code    int
	}
	// ErrorResponse is the struct that is eventually serialized as a negative response back to the user.
	ErrorResponse struct {
		error `json:"-" swaggerignore:"true"`
		Data  map[string]any `json:"data,omitempty"`
		Error string         `json:"error" example:"something is missing"`
		Code  string         `json:"code,omitempty" example:"SOMETHING_NOT_FOUND"`
	}
	Config struct {
		HTTPServer struct {
			CertPath string `yaml:"certPath"`
			KeyPath  string `yaml:"keyPath"`
			Port     uint16 `yaml:"port"`
		} `yaml:"httpServer"`
		DefaultEndpointTimeout time.Duration `yaml:"defaultEndpointTimeout"`
	}
)

// Private API.

const (
	json requestBinding = iota
	uri
	query
	header
	formMultipart

	languageHeader = "X-Language"
)

const (
	requestingUserIDCtxValueKey = "requestingUserIDCtxValueKey"

	authClientCtxValueKey = "authClientCtxValueKey"

	streamDefaultCapacity = 1000
)

var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	development bool
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	cfg Config
)

type (
	healthCheck struct {
		_ struct{} `allowUnauthorized:"true"` //nolint:revive // It's processed by the router.
	}
	requestBinding uint8
	// | srv is the internal representation of everything needed to bootstrap the http server.
	srv struct {
		State
		server             *http.Server
		router             *Router
		quit               chan<- os.Signal
		swaggerRoot        string
		nginxPrefix        string
		applicationYAMLKey string
	}
)
