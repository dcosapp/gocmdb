package utils

import "gopkg.in/gomail.v2"

type Email struct {
	smtpServer string
	smtpPort   int
	user       string
	password   string
}

func NewEmail(smtpServer string, smtpPort int, user, password string) *Email {
	return &Email{
		smtpServer: smtpServer,
		smtpPort:   smtpPort,
		user:       user,
		password:   password,
	}
}

func (e *Email) Send(to []string, subject string, msg string, attaches []string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", e.user)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", msg)

	for _, attach := range attaches {
		m.Attach(attach)
	}

	d := gomail.NewDialer(e.smtpServer, e.smtpPort, e.user, e.password)

	return d.DialAndSend(m)
}
