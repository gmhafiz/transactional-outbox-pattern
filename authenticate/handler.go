package authenticate

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"

	"transactional-outbox-pattern/mail"
	"transactional-outbox-pattern/task"
)

type Handler struct {
	db   *sqlx.DB
	mail *mail.SMTP
}

func New(db *sqlx.DB, mail *mail.SMTP) *Handler {
	return &Handler{
		db:   db,
		mail: mail,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req MailRequest
	err := render.DecodeJSON(r.Body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err = h.send(ctx, &req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// send writes to both mail_deliveries and outbox table in one transaction
func (h *Handler) send(ctx context.Context, emailRequest *MailRequest) error {
	tx, err := h.db.Begin()
	if err != nil {
		log.Println(err)
		return err
	}

	// defer is a special Go keyword that runs the following statement upon exiting a function or
	// method. This ensures a database roll back is executed if an error occurs.
	// Committing a transaction does not roll back a transaction.
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)

	email := h.mail.Build(&mail.Content{
		Sender:    emailRequest.From,
		To:        emailRequest.To,
		Subject:   emailRequest.Subject,
		HTML:      "",
		Plaintext: emailRequest.Content,
	})

	msgPayload, err := json.Marshal(email)
	if err != nil {
		log.Println(err)
		return err
	}

	var mailDeliveryID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO mail_deliveries (created_at	, content)
		VALUES ($1, $2) RETURNING id
		`, time.Now(), msgPayload).Scan(&mailDeliveryID)
	if err != nil {
		log.Println(err)
		return err
	}

	// task.Message is the one gets passed into the queue
	t := task.Message{
		ID:   mailDeliveryID,
		Mail: email,
	}

	taskPayload, err := json.Marshal(t)
	if err != nil {
		log.Println(err)
		return err
	}

	id, err := uuid.NewV7()
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO outbox (id, type, payload)
		VALUES ($1, $2, $3)
		`, id, task.TypeEmailDelivery, taskPayload)
	if err != nil {
		log.Println(err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
