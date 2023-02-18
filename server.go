package server

import (
	"context"
	"net/http"
	"transactional-outbox-pattern/mail"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"

	"transactional-outbox-pattern/config"
	"transactional-outbox-pattern/queue"
	"transactional-outbox-pattern/task"
)

type Server struct {
	Cfg        *config.Config
	Router     *chi.Mux
	HTTPServer *http.Server

	DB    *sqlx.DB
	Redis *redis.Client

	Mail *mail.SMTP

	Asynq    *queue.Asynq
	Producer *task.Producer
}

func (s *Server) CloseResources(ctx context.Context) {
	_ = s.DB.Close()

	s.Redis.Shutdown(ctx)

	s.Asynq.Server.Shutdown()
	_ = s.Asynq.Client.Close()
}

func New() *Server {
	return &Server{
		Cfg:    config.New(),
		Router: chi.NewRouter(),
	}
}
