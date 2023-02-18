package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	server "transactional-outbox-pattern"
	"transactional-outbox-pattern/authenticate"
	"transactional-outbox-pattern/mail"
	"transactional-outbox-pattern/queue"
	"transactional-outbox-pattern/store"
	"transactional-outbox-pattern/task"
)

func main() {
	s := server.New()
	s.DB = store.Database(s.Cfg.Database)
	s.Redis = store.Redis(s.Cfg.Redis)
	s.Asynq = queue.New(s.Cfg.Redis)
	s.Mail = mail.New(s.Cfg.Mail)
	s.Producer = task.NewProducer(s.DB, s.Asynq)
	go s.Producer.Poll()
	s.Router = chi.NewRouter()

	h := authenticate.New(s.DB, s.Mail)

	s.Router.Route("/api/mail", func(router chi.Router) {
		router.Post("/send", h.Handle)
	})

	s.HTTPServer = &http.Server{
		Addr:              s.Cfg.Api.Host + ":" + s.Cfg.Api.Port,
		Handler:           s.Router,
		ReadHeaderTimeout: s.Cfg.Api.ReadHeaderTimeout,
	}

	go func() {
		start(s)
	}()

	_ = gracefulShutdown(context.Background(), s)
}

func start(s *server.Server) {
	log.Printf("Serving at %s:%s\n", s.Cfg.Api.Host, s.Cfg.Api.Port)
	err := s.HTTPServer.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func gracefulShutdown(ctx context.Context, s *server.Server) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	<-quit

	log.Println("Shutting down...")

	ctx, shutdown := context.WithTimeout(ctx, s.Cfg.Api.GracefulTimeout*time.Second)
	defer shutdown()

	s.CloseResources(ctx)

	return s.HTTPServer.Shutdown(ctx)
}
