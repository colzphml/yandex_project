// Package server создает экземпляр сервера.
package server

import (
	"context"
	"net"
	"net/http"

	"github.com/colzphml/yandex_project/internal/handlers"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	//"middleware" используется в 2 пакетах, потому для собственного алиас
	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	cgrpc "github.com/colzphml/yandex_project/internal/grpc"
	pb "github.com/colzphml/yandex_project/internal/metrics/proto"
	mdw "github.com/colzphml/yandex_project/internal/middleware"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var log = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "server").Logger()

func HTTPServer(ctx context.Context, cfg *serverutils.ServerConfig, repo storage.Repositorier) *http.Server {
	r := chi.NewRouter()
	r.Use(mdw.GzipHandle)
	r.Use(mdw.SubNet(cfg))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", handlers.SaveHandler(ctx, repo, cfg))
	r.Get("/value/{metric_type}/{metric_name}", handlers.GetValueHandler(ctx, repo))
	r.Route("/update", func(r chi.Router) {
		r.Use(mdw.RSAHandler(cfg))
		r.Post("/", handlers.SaveJSONHandler(ctx, repo, cfg))
	})
	r.Post("/updates/", handlers.SaveJSONArrayHandler(ctx, repo, cfg))
	r.Post("/value/", handlers.GetJSONValueHandler(ctx, repo, cfg))
	r.Get("/ping", handlers.PingHandler(ctx, repo, cfg))
	r.Get("/", handlers.ListMetricsHandler(ctx, repo, cfg))
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
	s := grpc.NewServer(grpc.UnaryInterceptor(mdw.SubNetGRPCInterceptor(cfg)))
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
