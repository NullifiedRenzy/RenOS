package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTOTPValidateGeneratedCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Unix(1234567890, 0)
	code, err := generateTOTPCode(secret, now.Unix()/totpPeriod)
	if err != nil {
		t.Fatal(err)
	}
	if !ValidateTOTP(secret, code, now) {
		t.Fatal("expected generated code to validate")
	}
	if !ValidateTOTP(secret, code[:3]+" "+code[3:], now) {
		t.Fatal("expected spaced code to validate")
	}
}

func TestTOTPRejectsBadCode(t *testing.T) {
	if ValidateTOTP("JBSWY3DPEHPK3PXP", "abcdef", time.Now()) {
		t.Fatal("expected non-numeric code to be rejected")
	}
	if ValidateTOTP("JBSWY3DPEHPK3PXP", "12345", time.Now()) {
		t.Fatal("expected short code to be rejected")
	}
}

func TestGenerateTOTPSecret(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) == 0 || strings.Contains(secret, "=") {
		t.Fatalf("unexpected secret format: %q", secret)
	}
}

func TestTOTPAuthURL(t *testing.T) {
	got := TOTPAuthURL("RenOS", "admin@example.test", "SECRET")
	for _, want := range []string{"otpauth://totp/RenOS:admin@example.test", "secret=SECRET", "issuer=RenOS", "digits=6"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q does not contain %q", got, want)
		}
	}
}
