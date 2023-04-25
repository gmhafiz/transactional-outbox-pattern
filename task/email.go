package task

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"

	"transactional-outbox-pattern/mail"
)

const (
	// TypeEmailDelivery is a unique string used to identify this particular task which is sending
	// an email. This unique string is also used by asynq to identify which ProcessTask() method to
	// call.
	TypeEmailDelivery = "email:deliver"
)

// Message is the payload that gets passed into the queue
type Message struct {

	// ID of the primary key in mail_deliveries table
	ID int

	// Mail is the actual payload of the struct
	Mail *mail.SMTP
}

// Worker is our own custom struct that contains dependencies. It has a single method that implements
// asynq.Handler interface
type Worker struct {
	db *sqlx.DB
}

// NewEmailProcessor is the registered handler of the asynq mux. It accepts any custom dependencies
// you need.
func NewEmailProcessor(db *sqlx.DB) *Worker {
	return &Worker{
		db: db,
	}
}

// ProcessTask implements asynq.Handler. Any error that occurs will be handled by asynq
// library which is to retry until a certain number of time before declaring it
// as a failed state.
func (w *Worker) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var msg Message
	if err := json.Unmarshal(task.Payload(), &msg); err != nil {
		log.Println(err)
		_ = w.saveError(ctx, msg.ID, err)
		return err
	}

	// There is no need to row-lock because each task popped from the queue is exclusive to its
	// respective worker.
	_, err := w.db.ExecContext(ctx, `
		UPDATE mail_deliveries
		SET status = $1, start_time = $2, updated_at = $3
		WHERE id = $4
		`, "Started", time.Now(), time.Now(), msg.ID)
	if err != nil {
		log.Println(err)
		_ = w.saveError(ctx, msg.ID, err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(1)*time.Minute)
	defer cancel()

	if err := msg.Mail.Send(); err != nil {
		log.Println(err)
		err = w.saveError(ctx, msg.ID, err)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	// (*) Anything after email is sent, do not return an error or else this particular task
	// will be retried. You have achieved exactly-once delivery but may be left with
	// inconsistent state between mail_deliveries.status with actual email being sent.

	_, err = w.db.ExecContext(ctx, `
		UPDATE mail_deliveries
		SET status = $1, end_time = $2, updated_at = $3
		WHERE id = $4
		`, "Success", time.Now(), time.Now(), msg.ID)
	if err != nil {
		_ = w.saveError(ctx, msg.ID, err)
		log.Printf("failed to update status to success for task %d\n", msg.ID)
		// Ditto (*), not returning an error to prevent replay!
		// Solution:
		//   1. Thus, for any mail_deliveries.status is 'failed', must reconcile with mail
		//      provider.
		//   2. Have a background process (just like Producer) that re-attempts to queue mail_deliveries
		//      records with failed status. If the queue rejects because that task has already been
		//      processed (either asynq.ErrTaskIDConflict or asynq.ErrDuplicateTask), attempt to update
		//      status from failed to status.
		//   3. Use Postgres' LISTEN/NOTIFY that triggers an api to re-attempts to queue
		//      mail_deliveries records with failed status. Just like (2) but we save on database
		//      resource from polling every second.
	}

	return nil
}

func (w *Worker) saveError(ctx context.Context, id int, err error) error {
	type error struct {
		Error string `json:"error"`
	}

	e := &error{Error: err.Error()}
	bytes, errMarshall := json.Marshal(e)
	if errMarshall != nil {
		return errors.Join(err, errMarshall)
	}

	_, errDB := w.db.ExecContext(ctx, `
		UPDATE mail_deliveries
		SET status = $1, errors = $2, updated_at = $3
		WHERE id = $4
		`, "Failed", bytes, time.Now(), id)
	if errDB != nil {
		log.Println(errDB)
		err = errors.Join(errDB)
	}

	return err
}
