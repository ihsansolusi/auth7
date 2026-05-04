package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
)

type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
	startTLS bool
}

func NewSMTPMailer(host string, port int, username, password, from string, startTLS bool) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		startTLS: startTLS,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	const op = "mailer.SMTPMailer.Send"

	body, err := buildMIMEMessage(m.from, to, subject, htmlBody)
	if err != nil {
		return fmt.Errorf("%s: build MIME: %w", op, err)
	}

	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	var auth smtp.Auth
	if m.username != "" && m.password != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	envelopeFrom := extractEmail(m.from)

	if m.startTLS {
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: m.host,
		})
		if err != nil {
			return fmt.Errorf("%s: dial TLS: %w", op, err)
		}
		defer tlsConn.Close()

		client, err := smtp.NewClient(tlsConn, m.host)
		if err != nil {
			return fmt.Errorf("%s: new SMTP client: %w", op, err)
		}
		defer client.Close()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("%s: auth: %w", op, err)
			}
		}

		if err := client.Mail(envelopeFrom); err != nil {
			return fmt.Errorf("%s: mail from: %w", op, err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("%s: rcpt to: %w", op, err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("%s: data: %w", op, err)
		}
		if _, err := w.Write(body); err != nil {
			return fmt.Errorf("%s: write body: %w", op, err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("%s: close data: %w", op, err)
		}
		return nil
	}

	return smtp.SendMail(addr, auth, envelopeFrom, []string{to}, body)
}

func extractEmail(from string) string {
	start := -1
	end := -1
	for i := 0; i < len(from); i++ {
		if from[i] == '<' {
			start = i + 1
		}
		if from[i] == '>' {
			end = i
			break
		}
	}
	if start >= 0 && end > start {
		return from[start:end]
	}
	return from
}

func buildMIMEMessage(from, to, subject, htmlBody string) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	header := make(textproto.MIMEHeader)
	header.Set("From", from)
	header.Set("To", to)
	header.Set("Subject", mime.QEncoding.Encode("utf-8", subject))
	header.Set("MIME-Version", "1.0")
	header.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", boundary))

	for k, v := range header {
		for _, vv := range v {
			buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, vv))
		}
	}
	buf.WriteString("\r\n")

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Type", "text/html; charset=utf-8")
	partHeader.Set("Content-Transfer-Encoding", "quoted-printable")

	partWriter, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, fmt.Errorf("create part: %w", err)
	}

	qpWriter := quotedprintable.NewWriter(partWriter)
	if _, err := qpWriter.Write([]byte(htmlBody)); err != nil {
		return nil, fmt.Errorf("write quoted-printable: %w", err)
	}
	if err := qpWriter.Close(); err != nil {
		return nil, fmt.Errorf("close quoted-printable: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}

	return buf.Bytes(), nil
}
