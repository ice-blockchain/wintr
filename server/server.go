// SPDX-License-Identifier: ice License 1.0

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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/ice-blockchain/wintr/auth"
	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(state State, cfgKey, swaggerRoot string) Server {
	appCfg.MustLoadFromKey(cfgKey, &cfg)
	appCfg.MustLoadFromKey("development", &development)

	return &srv{State: state, swaggerRoot: swaggerRoot, applicationYAMLKey: cfgKey}
}

func (s *srv) ListenAndServe(ctx context.Context, cancel context.CancelFunc) {
	s.Init(ctx, cancel)
	s.setupRouter() //nolint:contextcheck // Nope, we don't need it.
	s.setupServer(ctx)
	go s.startServer()
	s.wait(ctx)
	s.shutDown() //nolint:contextcheck // Nope, we want to gracefully shutdown on a different context.
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
	s.router.GET("health-check", RootHandler(func(ctx context.Context, _ *Request[healthCheck, map[string]string]) (*Response[map[string]string], *Response[ErrorResponse]) { //nolint:lll // .
		if err := s.State.CheckHealth(ctx); err != nil {
			return nil, Unexpected(errors.Wrapf(err, "health check failed"))
		}

		return OK(&map[string]string{"clientIp": "1.2.3.4"}), nil
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
	authClient := auth.New(ctx, s.applicationYAMLKey)

	s.server = &http.Server{ //nolint:gosec // Not an issue, each request has a deadline set by the handler; and we're behind a proxy.
		Addr:    fmt.Sprintf(":%v", cfg.HTTPServer.Port),
		Handler: s.router,
		BaseContext: func(_ net.Listener) context.Context {
			return context.WithValue(ctx, authClientCtxValueKey, authClient) //nolint:staticcheck,revive // .
		},
	}
}

func (s *srv) startServer() {
	defer log.Info("server stopped listening")
	log.Info(fmt.Sprintf("server started listening on %v...", cfg.HTTPServer.Port))

	isUnexpectedError := func(err error) bool {
		return err != nil &&
			!errors.Is(err, io.EOF) &&
			!errors.Is(err, http.ErrServerClosed)
	}

	if err := s.server.ListenAndServeTLS(cfg.HTTPServer.CertPath, cfg.HTTPServer.KeyPath); isUnexpectedError(err) {
		s.quit <- syscall.SIGTERM
		log.Error(errors.Wrap(err, "server.ListenAndServeTLS failed"))
	}
}

func (s *srv) wait(ctx context.Context) {
	quit := make(chan os.Signal, 1)
	s.quit = quit
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
