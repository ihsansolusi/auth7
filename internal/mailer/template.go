package mailer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
)

type templateData struct {
	Title    string
	Body     string
	Code     string
	RefURL   string
	Username string
	Password string
}

var verifyTpl = template.Must(template.New("verify").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
  <h2 style="color:#1a365d">{{.Title}}</h2>
  <p style="color:#4a5568">Klik tombol di bawah untuk verifikasi email Anda:</p>
  <p><a href="{{.RefURL}}" style="background:#2b6cb0;color:white;padding:12px 24px;text-decoration:none;border-radius:6px;display:inline-block">Verifikasi Email</a></p>
  <p style="color:#a0aec0;font-size:12px;margin-top:32px">Auth7 — Core7 Identity Platform</p>
</body></html>`))

var resetTpl = template.Must(template.New("reset").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
  <h2 style="color:#1a365d">{{.Title}}</h2>
  <p style="color:#4a5568">Klik tombol di bawah untuk reset password Anda:</p>
  <p><a href="{{.RefURL}}" style="background:#2b6cb0;color:white;padding:12px 24px;text-decoration:none;border-radius:6px;display:inline-block">Reset Password</a></p>
  <p style="color:#a0aec0;font-size:12px;margin-top:32px">Auth7 — Core7 Identity Platform</p>
</body></html>`))

var otpTpl = template.Must(template.New("otp").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
  <h2 style="color:#1a365d">{{.Title}}</h2>
  <p style="color:#4a5568">Kode verifikasi Anda:</p>
  <p style="font-size:32px;font-weight:bold;color:#1a365d;letter-spacing:8px;text-align:center;padding:16px;background:#f7fafc;border-radius:8px">{{.Code}}</p>
  <p style="color:#4a5568">Kode berlaku 10 menit. Jangan bagikan ke siapapun.</p>
  <p style="color:#a0aec0;font-size:12px;margin-top:32px">Auth7 — Core7 Identity Platform</p>
</body></html>`))

func RenderVerificationEmail(title, verifyURL string) (string, error) {
	var buf bytes.Buffer
	if err := verifyTpl.Execute(&buf, templateData{Title: title, RefURL: verifyURL}); err != nil {
		return "", fmt.Errorf("render verify template: %w", err)
	}
	return buf.String(), nil
}

func RenderResetEmail(title, resetURL string) (string, error) {
	var buf bytes.Buffer
	if err := resetTpl.Execute(&buf, templateData{Title: title, RefURL: resetURL}); err != nil {
		return "", fmt.Errorf("render reset template: %w", err)
	}
	return buf.String(), nil
}

func RenderOTPEmail(title, code string) (string, error) {
	var buf bytes.Buffer
	if err := otpTpl.Execute(&buf, templateData{Title: title, Code: code}); err != nil {
		return "", fmt.Errorf("render OTP template: %w", err)
	}
	return buf.String(), nil
}

var newAccountTpl = template.Must(template.New("newaccount").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
  <h2 style="color:#1a365d">{{.Title}}</h2>
  <p style="color:#4a5568">Akun Anda telah dibuat. Gunakan kredensial berikut untuk masuk pertama kali:</p>
  <p style="color:#4a5568">Username: <b>{{.Username}}</b></p>
  <p style="color:#4a5568">Password sementara:</p>
  <p style="font-size:20px;font-weight:bold;color:#1a365d;letter-spacing:2px;text-align:center;padding:16px;background:#f7fafc;border-radius:8px;font-family:monospace">{{.Password}}</p>
  <p style="color:#4a5568">Anda akan diminta mengganti password ini saat login pertama.</p>
  <p style="color:#a0aec0;font-size:12px;margin-top:32px">Auth7 — Core7 Identity Platform</p>
</body></html>`))

func RenderNewAccountEmail(title, username, tempPassword string) (string, error) {
	var buf bytes.Buffer
	if err := newAccountTpl.Execute(&buf, templateData{Title: title, Username: username, Password: tempPassword}); err != nil {
		return "", fmt.Errorf("render new account template: %w", err)
	}
	return buf.String(), nil
}

type NoopMailer struct{}

func NewNoopMailer() *NoopMailer {
	return &NoopMailer{}
}

func (n *NoopMailer) Send(_ context.Context, _, _, _ string) error {
	return nil
}
