package provider

import "context"

type EmailRequest struct {
	To      string
	Subject string
	Body    string
}

type EmailSender interface {
	Send(ctx context.Context, req EmailRequest) error
}
