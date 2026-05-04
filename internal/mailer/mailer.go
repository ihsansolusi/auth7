package mailer

import "context"

type Mailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}
