package mailer

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

// MailClient describes a client that is able to send mails.
type MailClient interface {
	Send(from string, tos []string, subject string, body string, ccs, bccs []string) error
}

// LogMailer logs all messages using the given logger.
type LogMailer struct {
	logger *log.Logger
}

// ensure LogMailer implements the MailClient interface.
var _ MailClient = &LogMailer{}

// receive the pointer to a new LogMailer.
func NewLogMailer(logger *log.Logger) *LogMailer {
	return &LogMailer{
		logger: logger,
	}
}

// Send logs the given message using the client's logger.
func (m LogMailer) Send(from string, tos []string, subject string, body string, ccs, bccs []string) error {
	m.logger.Println("sending mail")
	m.logger.Printf("from: %s", from)
	m.logger.Printf("to: %s", strings.Join(tos, ", "))
	m.logger.Printf("cc: %s", strings.Join(ccs, ", "))
	m.logger.Printf("bcc: %s", strings.Join(bccs, ", "))
	m.logger.Printf("subject: %s", subject)
	m.logger.Printf("body: %s", body)
	return nil
}

// TLSMailer implements the MailClient interface.
type TLSMailer struct {
	host     string
	port     string
	username string
	password string
}

// ensure TLSMailer implements the MailClient interface.
var _ MailClient = &TLSMailer{}

// receive the pointer to a new TLSMailer.
func NewTLSMailer(host, port, username, password string) (*TLSMailer, error) {
	return &TLSMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
	}, nil
}

// Send creates a TLS encrypted SMTP connection to the configured host and port
// and sends the mail message.
func (m TLSMailer) Send(from string, tos []string, subject string, body string, ccs, bccs []string) error {
	if from == "" {
		from = m.username
	}
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(tos, ", ")
	headers["Cc"] = strings.Join(ccs, ", ")
	// BCCs should not be in headers
	// headers["Bcc"] = strings.Join(bccs, ", ")
	headers["Subject"] = subject
	// compose message
	var message string
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// create tcp connection
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", m.host, m.port))
	if err != nil {
		return fmt.Errorf("error while creating tcp connection to host %s with port %s: %w", m.host, m.port, err)
	}
	// create smtp client
	client, err := smtp.NewClient(conn, fmt.Sprintf("%s:%s", m.host, m.port))
	if err != nil {
		return fmt.Errorf("error while creating smtp client for host %s with port %s: %w", m.host, m.port, err)
	}
	// "upgrade" connection to tls
	tlsConfig := &tls.Config{
		ServerName: m.host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("unable to initialize StartTLS: %w", err)
	}
	// login auth
	auth := LoginAuth(m.username, m.password)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("error while creating smtp client authentication: %w", err)
	}
	// set Mail (from address)
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("unable to set from address %s: %w", from, err)
	}
	// set RCPT (all recipients)
	for _, recipients := range [][]string{tos, ccs, bccs} {
		for _, r := range recipients {
			if err := client.Rcpt(r); err != nil {
				return fmt.Errorf("unable to add recipient %s: %w", r, err)
			}
		}
	}
	// create writer for client data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("error while creating writer for smtp client data: %w", err)
	}
	// write message (headers + body)
	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("error while writing to smtp client data: %w", err)
	}
	// close writer & quit smtp client
	if err := w.Close(); err != nil {
		return fmt.Errorf("error while closing writer for smtp client data: %w", err)
	}
	client.Quit()
	return nil
}
