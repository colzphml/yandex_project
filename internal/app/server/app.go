// Package server создает экземпляр сервера.
package server

import (
	"context"
	"net"
	"net/http"

	"github.com/colzphml/yandex_project/internal/scenarios/handlers"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	pb "github.com/colzphml/yandex_project/internal/metrics/proto"
	"github.com/colzphml/yandex_project/internal/middleware"
	cgrpc "github.com/colzphml/yandex_project/internal/scenarios/grpc"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

var log = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "server").Logger()

func HTTPServer(ctx context.Context, cfg *serverutils.ServerConfig, repo storage.Repositorier) *http.Server {
	h := handlers.New(ctx, repo, cfg)
	r := chi.NewRouter()
	r.Use(middleware.GzipHandle)
	r.Use(middleware.SubNet(cfg))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", h.SaveHandler)
	r.Get("/value/{metric_type}/{metric_name}", h.GetValueHandler)
	r.Route("/update", func(r chi.Router) {
		r.Use(middleware.RSAHandler(cfg))
		r.Post("/", h.SaveJSONHandler)
	})
	r.Post("/updates/", h.SaveJSONArrayHandler)
	r.Post("/value/", h.GetJSONValueHandler)
	r.Get("/ping", h.PingHandler)
	r.Get("/", h.ListMetricsHandler)
	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed initialize server")
		}
	}()
	return srv
}

func GRPCServer(ctx context.Context, cfg *serverutils.ServerConfig, repo storage.Repositorier) *grpc.Server {
	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		log.Fatal().Err(err).Msg("failed initialize gRPC server")
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(middleware.SubNetGRPCInterceptor(cfg)))
	pb.RegisterMetricsServer(s, &cgrpc.MetricsServer{
		Cfg:  cfg,
		Repo: repo,
	})
	go func() {
		if err := s.Serve(listen); err != nil && err != grpc.ErrServerStopped {
			log.Fatal().Err(err).Msg("failed initialize server")
		}
	}()
	return s
}
