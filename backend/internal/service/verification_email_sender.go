package service

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
)

type SMTPVerificationEmailSender struct {
	cfg config.EmailConfig
}

func NewSMTPVerificationEmailSender(cfg config.EmailConfig) *SMTPVerificationEmailSender {
	return &SMTPVerificationEmailSender{cfg: cfg.WithDefaults()}
}

func (s *SMTPVerificationEmailSender) SendVerificationEmail(ctx context.Context, to, locale, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cfg := s.cfg.WithDefaults()
	if !cfg.Enabled() {
		return nil
	}

	authURL, err := verificationURL(cfg.VerificationBaseURL, token)
	if err != nil {
		return err
	}

	message := verificationMessage(cfg.From, to, locale, authURL)
	addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))

	var auth smtp.Auth
	if cfg.SMTPUsername != "" && cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(message))
}

func verificationURL(baseURL, token string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	query := parsedURL.Query()
	query.Set("token", token)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func verificationMessage(from, to, locale, authURL string) string {
	body := verificationBody(locale, authURL)
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: [MonthlyGoalTracker] Verify your email",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}

	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}

func verificationBody(locale, authURL string) string {
	if strings.EqualFold(locale, "en") {
		return fmt.Sprintf("Open this link to verify your MonthlyGoalTracker account:\n\n%s\n\nIf you did not request this email, you can ignore it.", authURL)
	}

	return fmt.Sprintf("MonthlyGoalTracker 계정을 인증하려면 아래 링크를 여세요.\n\n%s\n\n요청한 적이 없다면 이 메일을 무시해도 됩니다.", authURL)
}
