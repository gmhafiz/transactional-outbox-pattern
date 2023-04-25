package main

import (
	"log"

	"github.com/hibiken/asynq"

	server "transactional-outbox-pattern"
	"transactional-outbox-pattern/queue"
	"transactional-outbox-pattern/store"
	"transactional-outbox-pattern/task"
)

func main() {
	s := server.New()
	s.DB = store.Database(s.Cfg.Database)
	s.Asynq = queue.New(s.Cfg.Redis)

	mux := asynq.NewServeMux()

	// we inject the database as a dependency into the handler
	mux.Handle(task.TypeEmailDelivery, task.NewEmailProcessor(s.DB))

	if err := s.Asynq.Server.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
