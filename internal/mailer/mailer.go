package mailer

import (
	"github.com/rbicker/gooser/internal/store"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Messenger describes the functions to deliver application specific messages.
type Messenger interface {
	SendConfirmToken(user *store.User) error
	SendPasswordResetToken(user *store.User) error
}

// Mailer implements the Messenger interface.
type Mailer struct {
	mailClient       MailClient
	confirmUrl       string
	resetPasswordUrl string
	from             string
	siteName         string
}

// ensure Mailer implements the Messenger interface.
var _ Messenger = &Mailer{}

// NewMailer creates a new mailer and returns the pointer.
func NewMailer(mailClient MailClient, mailFrom, siteName, confirmUrl, resetPasswordUrl string) (*Mailer, error) {
	return &Mailer{
		mailClient: mailClient,
		confirmUrl: confirmUrl,
		from:       mailFrom,
		siteName:   siteName,
	}, nil
}

// SendConfirmToken sends the confirmation token.
func (m Mailer) SendConfirmToken(user *store.User) error {
	printer := message.NewPrinter(language.Make(user.Language))
	link := m.confirmUrl + "?token=" + user.ConfirmToken
	err := m.mailClient.Send(
		m.from,
		[]string{user.Mail},
		printer.Sprintf("%s: confirm mail address", m.siteName),
		printer.Sprintf("Hi %s! Please confirm your mail address by clicking the following link. Thanks!\n%s", user.Username, link),
		nil,
		nil,
	)
	if err != nil {
		status.Errorf(codes.Internal, printer.Sprintf("error while sending mail: %s", err))
	}
	return nil
}

// SendConfirmToken sends the confirmation token.
func (m Mailer) SendPasswordResetToken(user *store.User) error {
	printer := message.NewPrinter(language.Make(user.Language))
	link := m.resetPasswordUrl + "?token=" + user.PasswordResetToken
	err := m.mailClient.Send(
		m.from,
		[]string{user.Mail},
		printer.Sprintf("%s: password reset", m.siteName),
		printer.Sprintf("Hi %s! To reset your password, click the following link: \n%s\n\nIf you did not request to reset your password, please ignore this message. Thanks", user.Username, link),
		nil,
		nil,
	)
	if err != nil {
		status.Errorf(codes.Internal, printer.Sprintf("error while sending mail: %s", err))
	}
	return nil
}
