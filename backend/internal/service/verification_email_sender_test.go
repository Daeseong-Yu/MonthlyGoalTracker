package service

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

func TestPasswordResetURLPlacesTokenInFragment(t *testing.T) {
	resetURL, err := passwordResetURL("https://app.example.com/reset-password?locale=ko", "raw-token")
	if err != nil {
		t.Fatalf("expected reset URL, got %v", err)
	}

	parsedURL, err := url.Parse(resetURL)
	if err != nil {
		t.Fatalf("expected valid reset URL, got %v", err)
	}

	if parsedURL.Query().Get("resetToken") != "" {
		t.Fatal("expected reset token to be absent from query")
	}
	if parsedURL.Query().Get("locale") != "ko" {
		t.Fatalf("expected existing query params to be preserved, got %q", parsedURL.RawQuery)
	}

	fragment, err := url.ParseQuery(parsedURL.Fragment)
	if err != nil {
		t.Fatalf("expected query-style reset URL fragment, got %v", err)
	}
	if fragment.Get("resetToken") != "raw-token" {
		t.Fatalf("expected reset token in fragment, got %q", parsedURL.Fragment)
	}
}

func TestSESVerificationEmailSenderSendsVerificationEmail(t *testing.T) {
	client := &fakeSESEmailClient{}
	sender := newSESVerificationEmailSenderWithClient(testSESEmailConfig(), client)

	if err := sender.SendVerificationEmail(context.Background(), "owner@example.com", "en", "raw-token"); err != nil {
		t.Fatalf("expected verification email to send, got %v", err)
	}

	input := client.input
	if input == nil {
		t.Fatal("expected SES SendEmail input")
	}
	if input.FromEmailAddress == nil || *input.FromEmailAddress != "no-reply@example.com" {
		t.Fatalf("expected configured from address, got %v", input.FromEmailAddress)
	}
	if len(input.Destination.ToAddresses) != 1 || input.Destination.ToAddresses[0] != "owner@example.com" {
		t.Fatalf("expected destination address, got %v", input.Destination.ToAddresses)
	}
	if input.Content.Simple.Subject.Data == nil || *input.Content.Simple.Subject.Data != verificationSubject() {
		t.Fatalf("expected verification subject, got %v", input.Content.Simple.Subject.Data)
	}
	if input.Content.Simple.Body.Text.Data == nil || !strings.Contains(*input.Content.Simple.Body.Text.Data, "https://app.example.com/?token=raw-token") {
		t.Fatalf("expected verification token URL in body, got %v", input.Content.Simple.Body.Text.Data)
	}
}

func TestSESVerificationEmailSenderSendsPasswordResetEmail(t *testing.T) {
	client := &fakeSESEmailClient{}
	sender := newSESVerificationEmailSenderWithClient(testSESEmailConfig(), client)

	if err := sender.SendPasswordResetEmail(context.Background(), "owner@example.com", "ko", "reset-token"); err != nil {
		t.Fatalf("expected password reset email to send, got %v", err)
	}

	input := client.input
	if input == nil {
		t.Fatal("expected SES SendEmail input")
	}
	if input.Content.Simple.Subject.Data == nil || *input.Content.Simple.Subject.Data != passwordResetSubject() {
		t.Fatalf("expected password reset subject, got %v", input.Content.Simple.Subject.Data)
	}
	if input.Content.Simple.Body.Text.Data == nil || !strings.Contains(*input.Content.Simple.Body.Text.Data, "https://app.example.com/#resetToken=reset-token") {
		t.Fatalf("expected password reset token fragment in body, got %v", input.Content.Simple.Body.Text.Data)
	}
}

func testSESEmailConfig() config.EmailConfig {
	return config.EmailConfig{
		Provider:            config.EmailProviderSES,
		From:                "no-reply@example.com",
		SESRegion:           "us-east-1",
		VerificationBaseURL: "https://app.example.com/",
	}
}

type fakeSESEmailClient struct {
	input *sesv2.SendEmailInput
}

func (c *fakeSESEmailClient) SendEmail(ctx context.Context, input *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.input = input
	return &sesv2.SendEmailOutput{}, nil
}
