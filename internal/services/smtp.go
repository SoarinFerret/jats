package services

import (
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/soarinferret/jats/internal/config"
	"github.com/soarinferret/jats/internal/models"
)

type SMTPService struct {
	config *config.EmailConfig
}

func NewSMTPService(config *config.EmailConfig) *SMTPService {
	return &SMTPService{config: config}
}

func (s *SMTPService) SendTaskNotification(task *models.Task, subscribers []models.TaskSubscriber, subject, content string) error {
	if len(subscribers) == 0 {
		return nil
	}

	// Build recipient list
	var recipients []string
	for _, subscriber := range subscribers {
		recipients = append(recipients, subscriber.Email)
	}

	return s.sendEmail(recipients, subject, content, task.EmailMessageID)
}

func (s *SMTPService) SendTaskUpdate(task *models.Task, subscribers []models.TaskSubscriber, comment *models.Comment) error {
	if len(subscribers) == 0 {
		return nil
	}

	subject := fmt.Sprintf("Re: %s", task.Name)
	content := s.buildTaskUpdateContent(task, comment)

	var recipients []string
	for _, subscriber := range subscribers {
		recipients = append(recipients, subscriber.Email)
	}

	return s.sendEmail(recipients, subject, content, task.EmailMessageID)
}

func (s *SMTPService) sendEmail(recipients []string, subject, content, inReplyTo string) error {
	if s.config.SMTPHost == "" || s.config.FromEmail == "" {
		return fmt.Errorf("SMTP not configured")
	}

	// Setup authentication
	var auth smtp.Auth
	if s.config.SMTPAuth {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	// Build email message
	from := mail.Address{Name: s.config.FromName, Address: s.config.FromEmail}
	msg := s.buildMessage(from, recipients, subject, content, inReplyTo)

	// Send email
	if s.config.SMTPUseTLS {
		return s.sendWithTLS(auth, recipients, msg)
	} else {
		return smtp.SendMail(
			fmt.Sprintf("%s:%s", s.config.SMTPHost, s.config.SMTPPort),
			auth,
			s.config.FromEmail,
			recipients,
			[]byte(msg),
		)
	}
}

func (s *SMTPService) sendWithTLS(auth smtp.Auth, recipients []string, msg string) error {
	// Connect to the SMTP Server
	servername := fmt.Sprintf("%s:%s", s.config.SMTPHost, s.config.SMTPPort)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: s.config.SMTPInsecure,
		ServerName:         s.config.SMTPHost,
	}

	// Connect to server
	c, err := smtp.Dial(servername)
	if err != nil {
		return err
	}
	defer c.Close()

	// Start TLS
	if err = c.StartTLS(tlsconfig); err != nil {
		return err
	}

	// Auth if configured
	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return err
		}
	}

	// Set sender
	if err = c.Mail(s.config.FromEmail); err != nil {
		return err
	}

	// Set recipients
	for _, recipient := range recipients {
		if err = c.Rcpt(recipient); err != nil {
			return err
		}
	}

	// Send message
	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}

func (s *SMTPService) buildMessage(from mail.Address, recipients []string, subject, content, inReplyTo string) string {
	var msg strings.Builder

	// Headers
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(recipients, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	if inReplyTo != "" {
		msg.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", inReplyTo))
		msg.WriteString(fmt.Sprintf("References: %s\r\n", inReplyTo))
	}

	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	msg.WriteString("\r\n")

	// Body
	msg.WriteString(content)

	return msg.String()
}

func (s *SMTPService) buildTaskUpdateContent(task *models.Task, comment *models.Comment) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("Task: %s\n", task.Name))
	content.WriteString(fmt.Sprintf("Status: %s\n", task.Status))

	if task.Priority != "" {
		content.WriteString(fmt.Sprintf("Priority: %s\n", task.Priority))
	}

	if len(task.Tags) > 0 {
		content.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(task.Tags, ", ")))
	}

	content.WriteString("\n")

	if comment != nil {
		content.WriteString("New Comment:\n")
		content.WriteString(comment.Content)
		content.WriteString("\n")
	}

	return content.String()
}
