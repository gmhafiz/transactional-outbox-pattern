package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"transactional-outbox-pattern/config"
	"transactional-outbox-pattern/mail"
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
	_ = s.Asynq.Client.Close()
	s.Asynq.Server.Shutdown()

	_ = s.DB.Close()
	s.Redis.Shutdown(ctx)

}

func New() *Server {
	return &Server{
		Cfg:    config.New(),
		Router: chi.NewRouter(),
	}
}
