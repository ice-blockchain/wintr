// SPDX-License-Identifier: ice License 1.0

package wsserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"github.com/ice-blockchain/wintr/wsserver/internal/http2"
	"github.com/ice-blockchain/wintr/wsserver/internal/http3"
)

func New(service Service, cfgKey string) Server {
	var cfg internal.Config
	appcfg.MustLoadFromKey(cfgKey, &cfg)
	s := &srv{cfg: &cfg, service: service}
	s.setupRouter()
	s.h3server = http3.New(s.cfg, s.service, s.router)
	s.wsServer = http2.New(s.cfg, s.service, s.router)

	return s
}

func (s *srv) setupRouter() {
	if !s.cfg.Development {
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
	s.service.RegisterRoutes(s.router)
	log.Info(fmt.Sprintf("%v routes registered", len(s.router.Routes())))
}

func (s *srv) ListenAndServe(ctx context.Context, cancel context.CancelFunc) {
	s.service.Init(ctx, cancel)
	go s.startServer(ctx, s.h3server)
	go s.startServer(ctx, s.wsServer)
	s.wait(ctx)
	s.shutDown() //nolint:contextcheck // Nope, we want to gracefully shutdown on a different context.
}

func (s *srv) startServer(ctx context.Context, server internal.Server) {
	defer log.Info("server stopped listening")
	log.Info(fmt.Sprintf("server started listening on %v...", s.cfg.WSServer.Port))

	isUnexpectedError := func(err error) bool {
		return err != nil &&
			!errors.Is(err, io.EOF) &&
			!errors.Is(err, http.ErrServerClosed)
	}

	if err := server.ListenAndServeTLS(ctx, s.cfg.WSServer.CertPath, s.cfg.WSServer.KeyPath); isUnexpectedError(err) {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.shutdownServer(ctx, s.h3server)
	go s.shutdownServer(ctx, s.wsServer)

	if err := s.service.Close(ctx); err != nil && !errors.Is(err, io.EOF) {
		log.Error(errors.Wrap(err, "state close failed"))
	} else {
		log.Info("state close succeeded")
	}
}

func (*srv) shutdownServer(ctx context.Context, server internal.Server) {
	log.Info("shutting down server...")

	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, io.EOF) {
		log.Error(errors.Wrap(err, "server shutdown failed"))
	} else {
		log.Info("server shutdown succeeded")
	}
}
