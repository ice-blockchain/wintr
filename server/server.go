// SPDX-License-Identifier: BUSL-1.1

package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

	appCfg "github.com/ICE-Blockchain/wintr/config"
	"github.com/ICE-Blockchain/wintr/log"
)

func New(state State, cfgKey, swaggerRoot string) Server {
	appCfg.MustLoadFromKey(cfgKey, &cfg)
	appCfg.MustLoadFromKey("development", &development)

	return &srv{State: state, swaggerRoot: swaggerRoot}
}

func (s *srv) ListenAndServe(ctx context.Context, cancel context.CancelFunc) {
	s.Init(ctx, cancel)
	s.setupRouter()
	s.setupServer(ctx)
	go s.startServer(cancel)
	s.wait(ctx)
	s.shutDown()
}

func (s *srv) setupRouter() {
	if !development {
		gin.SetMode(gin.ReleaseMode)
		s.router = gin.New()
		s.router.Use(gin.Recovery())
	} else {
		gin.ForceConsoleColor()
		s.router = gin.Default()
	}
	log.Info(fmt.Sprintf("GIN Mode: %v\n", gin.Mode()))
	s.router.RemoteIPHeaders = []string{"cf-connecting-ip", "X-Real-IP", "X-Forwarded-For"}
	s.router.TrustedPlatform = gin.PlatformCloudflare
	s.router.HandleMethodNotAllowed = true
	s.router.RedirectFixedPath = true
	s.router.RemoveExtraSlash = true
	s.router.UseRawPath = true

	log.Info("registering routes...")
	s.RegisterRoutes(s.router)
	log.Info(fmt.Sprintf("%v routes registered", len(s.router.Routes())))
	s.setupSwaggerRoutes()
	s.setupHealthCheckRoutes()
}

func (s *srv) setupHealthCheckRoutes() {
	s.router.GET("health-check", RootHandler(NewRequestCheckHealth, func(ctx context.Context, request ParsedRequest) Response {
		return s.State.CheckHealth(ctx, request.(*RequestCheckHealth))
	}))
}

func (s *srv) setupSwaggerRoutes() {
	root := s.swaggerRoot
	if root == "" {
		return
	}
	s.router.
		GET(root, func(c *gin.Context) {
			c.Redirect(http.StatusFound, (&url.URL{Path: fmt.Sprintf("%v/swagger/index.html", root)}).RequestURI())
		}).
		GET(fmt.Sprintf("%v/swagger/*any", root), ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (s *srv) setupServer(ctx context.Context) {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.HTTPServer.Port),
		Handler: s.router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}

func (s *srv) startServer(cancel context.CancelFunc) {
	defer cancel()
	defer log.Info("server stopped listening")
	log.Info(fmt.Sprintf("server started listening on %v...", cfg.HTTPServer.Port))

	isUnexpectedError := func(err error) bool {
		return err != nil &&
			!errors.Is(err, io.EOF) &&
			!errors.Is(err, http.ErrServerClosed)
	}

	if err := s.server.ListenAndServeTLS(cfg.HTTPServer.CertPath, cfg.HTTPServer.KeyPath); isUnexpectedError(err) {
		log.Error(errors.Wrap(err, "server.ListenAndServeTLS failed"))
	}
}

func (s *srv) wait(ctx context.Context) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-quit:
	}
}

func (s *srv) shutDown() {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DefaultEndpointTimeout)
	defer cancel()
	log.Info("shutting down server...")

	if err := s.server.Shutdown(ctx); err != nil && !errors.Is(err, io.EOF) {
		log.Error(errors.Wrap(err, "server shutdown failed"))
	} else {
		log.Info("server shutdown succeeded")
	}

	if err := s.State.Close(ctx); err != nil && !errors.Is(err, io.EOF) {
		log.Error(errors.Wrap(err, "state close failed"))
	} else {
		log.Info("state close succeeded")
	}
}

func NewRequestCheckHealth() ParsedRequest {
	return new(RequestCheckHealth)
}

func (r *RequestCheckHealth) Validate() *Response {
	return nil
}

func (r *RequestCheckHealth) Bindings(c *gin.Context) []func(obj interface{}) error {
	return []func(interface{}) error{ShouldBindClientIP(c)}
}

func (r *RequestCheckHealth) SetClientIP(ip net.IP) {
	if r.ClientIP == nil {
		r.ClientIP = ip
	}
}

func (r *RequestCheckHealth) GetClientIP() net.IP {
	return r.ClientIP
}
