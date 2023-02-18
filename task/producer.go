package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"

	"transactional-outbox-pattern/queue"
)

type outbox struct {
	ID      uuid.UUID `db:"id"`
	Type    string    `db:"type"`
	Payload []byte    `db:"payload"`
}

func NewProducer(db *sqlx.DB, asynq *queue.Asynq) *Producer {
	return &Producer{
		db:    db,
		asynq: asynq,
	}
}

type Producer struct {
	asynq *queue.Asynq
	db    *sqlx.DB
}

const interval = time.Duration(1)

// Poll continuously watches outbox table for any records.
func (p *Producer) Poll() {

	// Using a sleep can potentially reduce throughput if a loop runs for too long.
	// A ticker instead, ensures consistent and regular work.
	ticker := time.NewTicker(interval * time.Second)

	p.run()
	for range ticker.C {
		p.run()
	}
}

func (p *Producer) run() {
	ctx := context.Background()

	// We sandwich queuing inside a database transaction.
	tx, err := p.db.Begin()
	if err != nil {
		msg := fmt.Errorf("failed to start a transaction: %w", err)
		log.Println(msg)
		return
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)

	// See (A) to see potential issue.
	batch := 10

	// 1. Batch retrieval to reduce DB load.
	// 2. To only get unclaimed records, use ...FOR UPDATE SKIP LOCKED. This ensures the rows we are
	// 	  reading are locked. This prevents queries in other processes or producers to not select
	//	  these locked rows.
	//	  More info: https://www.enterprisedb.com/blog/what-skip-locked-postgresql-95
	// 3. We are also doing a SELECT, then DELETE once the transaction commits. This keeps the
	//	  number of records in outbox table small.
	rows, err := tx.QueryContext(ctx,
		`
				DELETE FROM outbox
				WHERE id IN (SELECT o.id
							 FROM outbox o
							 ORDER BY id
								 FOR UPDATE SKIP LOCKED
							 LIMIT $1)
				RETURNING *;`, batch)
	if err != nil {
		return
	}

	for rows.Next() {

		var record outbox
		if err := rows.Scan(&record.ID, &record.Type, &record.Payload); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Simply means no task to be queued yet.
				return
			}

			log.Println("other database scan error: %w", err)
			return
		}

		t := asynq.NewTask(record.Type, record.Payload,

			// To make each task idempotent, we assign our created unique ID to the queue.
			asynq.TaskID(record.ID.String()),

			// Hold task IDs for a period of time for asynq to check for uniqueness.
			// If the second identical task is pushed to asynq after the retention period is over,
			// we will get duplication of tasks. The longer retention, less chance of duplication,
			// but at the cost of using more memory holding the IDs.
			asynq.Retention(10*time.Minute),
		)
		_, err = p.asynq.Client.EnqueueContext(ctx, t)
		if err != nil {
			// If fails to enqueue, transaction is not committed which is good.
			// Reasons why it could fail:
			//  - redis is unavailable
			//  - redis not working
			//      - out of memory
			//  - network failure
			//
			// For these reasons, this loop will attempt to send the payload to asynq (redis) again.
			//
			// If it fails because of the task has done or in the middle or processing, we simply
			// delete that record from outbox table because it is already sent to the queue.
			if errors.Is(err, asynq.ErrTaskIDConflict) {
				log.Println(fmt.Errorf("preventing task duplication. task ID %s: %w", record.ID, err))

				if err = tx.Commit(); err != nil {
					log.Println("failed to commit")
					return
				}

				continue
			}
			if errors.Is(err, asynq.ErrDuplicateTask) {
				log.Println(fmt.Errorf("task is already running. task ID %s: %w", record.ID, err))

				if err = tx.Commit(); err != nil {
					log.Println("failed to commit")
					return
				}

				continue
			}

			// (A) We are doing ${batch} amount of tasks in one transaction. There is a risk of not
			// all task in the batch to queue.
			// Unless the error is related to asynq, we simply return which causes transaction to
			// rollback.
			// The next run() loop will pick up both sent and unsent tasks.
			//     - Queuing sent tasks will return either ErrTaskIDConflict or ErrDuplicateTask
			//     - Queuing failed tasks will go through.
			log.Println(fmt.Errorf("could not enqueue task ID %s: %w", record.ID, err))
			return
		}
	}

	// The act of COMMIT will delete the records.
	if err = tx.Commit(); err != nil {
		// If committing fails, returning will cause a rollback. And those locked rows will be
		// released for other processes or producers to pick up.
		log.Println("failed to commit")
		return
	}
}
