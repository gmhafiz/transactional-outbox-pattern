package mail

import (
	"fmt"
	"net/smtp"
	"strings"

	"transactional-outbox-pattern/config"
)

type SMTP struct {
	//Config *Config
	Config config.Mail

	Content *Content
}

type Content struct {
	Sender  string
	To      []string
	Subject string

	HTML      string
	Plaintext string
}

func New(cfg config.Mail) *SMTP {
	return &SMTP{
		Config: cfg,
	}
}

func (m *SMTP) Build(content *Content) *SMTP {
	if content.HTML != "" {
		msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
		msg += fmt.Sprintf("From: %s\r\n", content.Sender)
		msg += fmt.Sprintf("To: %s\r\n", strings.Join(content.To, ";"))
		msg += fmt.Sprintf("Subject: %s\r\n", content.Subject)
		msg += fmt.Sprintf("\r\n%s\r\n", content.HTML)
		content.HTML = msg
	}
	if content.Plaintext != "" {
		msg := "MIME-version: 1.0;\nContent-Type: text/plaintext; charset=\"UTF-8\";\r\n"
		msg += fmt.Sprintf("From: %s\r\n", content.Sender)
		msg += fmt.Sprintf("To: %s\r\n", strings.Join(content.To, ";"))
		msg += fmt.Sprintf("Subject: %s\r\n", content.Subject)
		msg += fmt.Sprintf("\r\n%s\r\n", content.Plaintext)
		content.Plaintext = msg
	}

	m.Content = content

	return m
}

func (m *SMTP) Send() error {
	auth := smtp.PlainAuth("", m.Config.Username, m.Config.Password, m.Config.Host)
	if err := smtp.SendMail(
		fmt.Sprintf("%s:%d", m.Config.Host, m.Config.Port),
		auth,
		m.Content.Sender,
		m.Content.To,
		[]byte(m.Content.Plaintext),
	); err != nil {
		return err
	}
	return nil
}
