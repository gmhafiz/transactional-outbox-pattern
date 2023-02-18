package queue

import (
	"fmt"

	"github.com/hibiken/asynq"

	"transactional-outbox-pattern/config"
)

type Asynq struct {
	redisClientOpt asynq.RedisClientOpt
	Client         *asynq.Client
	Server         *asynq.Server
}

const (
	Critical = "critical"
	Default  = "default"
	Low      = "low"
)

const (
	// Specify how many concurrent workers to use. Ideally the number is
	// number of threads + spindle count
	workerCount = 12

	// total number doesn't have to be equal to workerCount
	// these are ratios of work. So queueCriticalValue is done 50% of the time
	// (3 / (3+2+1) = 0.5
	queueCriticalValue = 3
	queueDefaultValue  = 2
	queueLowValue      = 1
)

func New(cfgRedis config.Redis) *Asynq {
	c := &Asynq{}

	cfg := asynq.Config{
		Concurrency: workerCount,
		// Optionally specify multiple queues with different priority.
		Queues: map[string]int{
			Critical: queueCriticalValue,
			Default:  queueDefaultValue,
			Low:      queueLowValue,
		},
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: fmt.Sprintf("%s:%s", cfgRedis.Host, cfgRedis.Port)},
		cfg,
	)

	c.Client = asynq.NewClient(c.redisClientOpt)
	c.Server = srv

	return c
}
