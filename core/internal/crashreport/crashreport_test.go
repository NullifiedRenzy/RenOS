package crashreport

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactEnvRemovesSecrets(t *testing.T) {
	env := map[string]string{
		"RENOS_JWT_SECRET":        "s3cret-jwt",
		"RENOS_ENCRYPTION_KEY":    "s3cret-enc",
		"DB_PASSWORD":             "hunter2",
		"OPENAI_API_KEY":          "sk-1234",
		"AWS_SECRET_ACCESS_KEY":   "aws-secret",
		"AUTH_TOKEN":              "tok",
		"RENOS_ADMIN_PASSWORD":    "admin-pw",
		"PATH":                    "/usr/bin:/bin",
		"HOME":                    "/root",
		"RENOS_DATABASE_URL":      "postgres://renos:pw@localhost:5432/renos",
		"RENOS_LOG_LEVEL":         "info",
		"RENOS_TELEMETRY_ENABLED": "false",
	}

	redacted := RedactEnv(env)

	for _, secretKey := range []string{"RENOS_JWT_SECRET", "RENOS_ENCRYPTION_KEY", "DB_PASSWORD", "OPENAI_API_KEY", "AWS_SECRET_ACCESS_KEY", "AUTH_TOKEN", "RENOS_ADMIN_PASSWORD"} {
		if v := redacted[secretKey]; v != "[REDACTED]" {
			t.Errorf("key %q: expected [REDACTED], got %q", secretKey, v)
		}
	}

	for _, safeKey := range []string{"PATH", "HOME", "RENOS_LOG_LEVEL", "RENOS_TELEMETRY_ENABLED"} {
		if redacted[safeKey] != env[safeKey] {
			t.Errorf("key %q should not be redacted: got %q", safeKey, redacted[safeKey])
		}
	}

	// The original map must not be mutated.
	if env["RENOS_JWT_SECRET"] != "s3cret-jwt" {
		t.Errorf("RedactEnv mutated the input map")
	}

	// Serialized output must not contain the raw secret values.
	serialized := mustJSON(redacted)
	for _, secret := range []string{"s3cret-jwt", "s3cret-enc", "hunter2", "sk-1234", "aws-secret", "tok", "admin-pw"} {
		if strings.Contains(string(serialized), secret) {
			t.Errorf("serialized env leaked %q", secret)
		}
	}
}

func TestBuildReportRedactsEnvironment(t *testing.T) {
	t.Setenv("RENOS_JWT_SECRET", "super-secret-value")
	t.Setenv("RENOS_LOG_LEVEL", "debug")

	rep := New(Options{App: "renos-core", Version: "1.0.0", CrashDir: t.TempDir()})
	report := rep.BuildReport("boom", []byte("stack trace here"))

	if report.Panic != "boom" {
		t.Errorf("expected panic value boom, got %q", report.Panic)
	}
	if report.Stack != "stack trace here" {
		t.Errorf("stack trace not propagated")
	}
	if report.Env["RENOS_JWT_SECRET"] != "[REDACTED]" {
		t.Errorf("environment secret was not redacted in report")
	}
	if report.Env["RENOS_LOG_LEVEL"] != "debug" {
		t.Errorf("non-sensitive env value was lost")
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if strings.Contains(string(raw), "super-secret-value") {
		t.Errorf("serialized report leaked the JWT secret")
	}
}

func TestWritePersistsReport(t *testing.T) {
	dir := t.TempDir()
	rep := New(Options{App: "renos-core", CrashDir: dir})
	report := rep.BuildReport("boom", []byte("stack"))

	crashDir, err := rep.Write(report)
	if err != nil {
		t.Fatalf("write report: %v", err)
	}

	for _, f := range []string{"report.json", "stack.txt", "goroutines.txt"} {
		if _, err := os.Stat(filepath.Join(crashDir, f)); err != nil {
			t.Errorf("expected %s to exist in crash dir", f)
		}
	}
}

func TestBundleReportCreatesValidTarball(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(logFile, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rep := New(Options{
		App:         "renos-core",
		CrashDir:    t.TempDir(),
		LogFile:     logFile,
		MaxLogLines: 2,
	})
	report := rep.BuildReport("boom", []byte("stack"))

	bundle, err := rep.BundleReport(report)
	if err != nil {
		t.Fatalf("bundle report: %v", err)
	}

	gz, err := gzip.NewReader(bytes.NewReader(bundle))
	if err != nil {
		t.Fatalf("bundle is not valid gzip: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	names := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar: %v", err)
		}
		data, _ := io.ReadAll(tr)
		names[hdr.Name] = string(data)
	}

	for _, want := range []string{"report.json", "stack.txt", "goroutines.txt", "logs.txt"} {
		if _, ok := names[want]; !ok {
			t.Errorf("bundle is missing %s", want)
		}
	}

	// Log tail must respect MaxLogLines.
	if !strings.Contains(names["logs.txt"], "line3") {
		t.Errorf("log tail is missing the final line")
	}
	if strings.Contains(names["logs.txt"], "line1") {
		t.Errorf("log tail exceeded MaxLogLines")
	}
}

func TestCapturePanicRepanicsAfterCapture(t *testing.T) {
	dir := t.TempDir()
	rep := New(Options{App: "renos-core", CrashDir: dir})

	defer func() {
		if rec := recover(); rec == nil {
			t.Errorf("CapturePanic must re-raise the original panic")
		}
	}()

	defer rep.CapturePanic()
	panic("test panic")
}
