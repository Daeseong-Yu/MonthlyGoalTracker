package service

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SMTPVerificationEmailSender struct {
	cfg config.EmailConfig
}

type SESVerificationEmailSender struct {
	cfg       config.EmailConfig
	mu        sync.Mutex
	client    sesEmailClient
	newClient func(context.Context, config.EmailConfig) (sesEmailClient, error)
}

type sesEmailClient interface {
	SendEmail(context.Context, *sesv2.SendEmailInput, ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

func NewSMTPVerificationEmailSender(cfg config.EmailConfig) *SMTPVerificationEmailSender {
	return &SMTPVerificationEmailSender{cfg: cfg.WithDefaults()}
}

func NewSESVerificationEmailSender(cfg config.EmailConfig) *SESVerificationEmailSender {
	return &SESVerificationEmailSender{
		cfg:       cfg.WithDefaults(),
		newClient: newSESEmailClient,
	}
}

func newSESVerificationEmailSenderWithClient(cfg config.EmailConfig, client sesEmailClient) *SESVerificationEmailSender {
	return &SESVerificationEmailSender{
		cfg:    cfg.WithDefaults(),
		client: client,
	}
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
	return s.sendPlainTextEmail(to, message)
}

func (s *SESVerificationEmailSender) SendVerificationEmail(ctx context.Context, to, locale, token string) error {
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

	return s.sendPlainTextEmail(ctx, to, verificationSubject(), verificationBody(locale, authURL))
}

func (s *SMTPVerificationEmailSender) SendPasswordResetEmail(ctx context.Context, to, locale, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cfg := s.cfg.WithDefaults()
	if !cfg.Enabled() {
		return nil
	}

	resetURL, err := passwordResetURL(cfg.PasswordResetBaseURL, token)
	if err != nil {
		return err
	}

	message := passwordResetMessage(cfg.From, to, locale, resetURL)
	return s.sendPlainTextEmail(to, message)
}

func (s *SESVerificationEmailSender) SendPasswordResetEmail(ctx context.Context, to, locale, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cfg := s.cfg.WithDefaults()
	if !cfg.Enabled() {
		return nil
	}

	resetURL, err := passwordResetURL(cfg.PasswordResetBaseURL, token)
	if err != nil {
		return err
	}

	return s.sendPlainTextEmail(ctx, to, passwordResetSubject(), passwordResetBody(locale, resetURL))
}

func (s *SMTPVerificationEmailSender) sendPlainTextEmail(to, message string) error {
	cfg := s.cfg.WithDefaults()
	addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))

	var auth smtp.Auth
	if cfg.SMTPUsername != "" && cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(message))
}

func (s *SESVerificationEmailSender) sendPlainTextEmail(ctx context.Context, to, subject, body string) error {
	cfg := s.cfg.WithDefaults()
	client, err := s.emailClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(cfg.From),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Text: &types.Content{
						Data:    aws.String(body),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	})

	return err
}

func (s *SESVerificationEmailSender) emailClient(ctx context.Context) (sesEmailClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s.client, nil
	}

	client, err := s.newClient(ctx, s.cfg.WithDefaults())
	if err != nil {
		return nil, err
	}

	s.client = client
	return s.client, nil
}

func newSESEmailClient(ctx context.Context, cfg config.EmailConfig) (sesEmailClient, error) {
	awsConfig, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(cfg.SESRegion))
	if err != nil {
		return nil, err
	}

	return sesv2.NewFromConfig(awsConfig), nil
}

func verificationURL(baseURL, token string) (string, error) {
	return emailActionURL(baseURL, "token", token)
}

func passwordResetURL(baseURL, token string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	fragment := url.Values{}
	fragment.Set("resetToken", token)
	parsedURL.Fragment = "resetToken=" + token
	parsedURL.RawFragment = fragment.Encode()

	return parsedURL.String(), nil
}

func emailActionURL(baseURL, tokenParam, token string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	query := parsedURL.Query()
	query.Set(tokenParam, token)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func verificationMessage(from, to, locale, authURL string) string {
	body := verificationBody(locale, authURL)
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + verificationSubject(),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}

	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}

func verificationSubject() string {
	return "[MonthlyGoalTracker] Verify your email"
}

func verificationBody(locale, authURL string) string {
	if strings.EqualFold(locale, "en") {
		return fmt.Sprintf("Open this link to verify your MonthlyGoalTracker account:\n\n%s\n\nIf you did not request this email, you can ignore it.", authURL)
	}

	return fmt.Sprintf("MonthlyGoalTracker 계정을 인증하려면 아래 링크를 여세요.\n\n%s\n\n요청한 적이 없다면 이 메일을 무시해도 됩니다.", authURL)
}

func passwordResetMessage(from, to, locale, resetURL string) string {
	body := passwordResetBody(locale, resetURL)
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + passwordResetSubject(),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}

	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}

func passwordResetSubject() string {
	return "[MonthlyGoalTracker] Reset your password"
}

func passwordResetBody(locale, resetURL string) string {
	if strings.EqualFold(locale, "en") {
		return fmt.Sprintf("Open this link to reset your MonthlyGoalTracker password:\n\n%s\n\nIf you did not request this email, you can ignore it.", resetURL)
	}

	return fmt.Sprintf("MonthlyGoalTracker 비밀번호를 재설정하려면 아래 링크를 여세요.\n\n%s\n\n요청한 적이 없다면 이 메일을 무시해도 됩니다.", resetURL)
}
