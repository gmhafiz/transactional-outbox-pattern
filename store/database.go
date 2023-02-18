package store

import (
	"fmt"
	"log"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"

	"transactional-outbox-pattern/config"
)

func Database(cfg config.Database) *sqlx.DB {

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Pass, cfg.Name)

	db, err := sqlx.Open(cfg.Driver, dsn)
	if err != nil {
		log.Fatal(err)
	}

	return db
}
