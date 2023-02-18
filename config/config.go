package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Api      Api
	Database Database
	Redis    Redis
	Mail     Mail
}

func New() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	return &Config{
		Api:      API(),
		Database: DataStore(),
		Redis:    NewRedis(),
		Mail:     NewMail(),
	}
}

type Api struct {
	Name              string        `default:"go8_api"`
	Host              string        `default:"0.0.0.0"`
	Port              string        `default:"3080"`
	ReadHeaderTimeout time.Duration `default:"60s"`

	GracefulTimeout time.Duration `default:"8s"`

	RequestLog bool `default:"false"`
	RunSwagger bool `default:"true"`
}

func API() Api {
	var api Api
	envconfig.MustProcess("API", &api)

	return api
}

type Database struct {
	Driver                 string        `default:"pgx"`
	Host                   string        `default:"0.0.0.0"`
	Port                   uint16        `default:"5432"`
	Name                   string        `default:"outbox_pattern"`
	User                   string        `default:"user"`
	Pass                   string        `default:"password"`
	SslMode                string        `default:"disable"`
	MaxConnectionPool      int           `default:"4"`
	MaxIdleConnections     int           `default:"4"`
	ConnectionsMaxLifeTime time.Duration `default:"300s"`
}

func DataStore() Database {
	var db Database
	envconfig.MustProcess("DB", &db)

	return db
}

type Redis struct {
	Host string `default:"0.0.0.0"`
	Port string `default:"6379"`
	Name int    `default:"1"`
	User string
	Pass string
}

func NewRedis() Redis {
	var cache Redis
	envconfig.MustProcess("REDIS", &cache)

	return cache
}

type Mail struct {
	Host     string `default:"localhost"`
	Port     int    `default:"1025"`
	Username string
	Password string
	//Encryption mail.EncryptionSTARTTLS

	Secret string
}

func NewMail() Mail {
	var mail Mail
	envconfig.MustProcess("MAIL", &mail)

	return mail
}
