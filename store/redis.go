package store

import (
	"fmt"

	"github.com/redis/go-redis/v9"

	"transactional-outbox-pattern/config"
)

func Redis(cfg config.Redis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Pass,
		DB:       cfg.Name,
	})
}
