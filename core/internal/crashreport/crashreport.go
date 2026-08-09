// Package crashreport captures, persists and bundles crash reports so that
// production failures are observable even when the process terminates.
//
// Reports contain process metadata, the panic value, a stack trace, a full
// goroutine dump and a SANITIZED copy of the process environment — any key
// that looks like a password, secret, key, token or auth material is redacted.
package crashreport

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DefaultDir is where crash reports are written when no directory is set.
const DefaultDir = "/var/lib/renos/crashes"

// sensitiveKeyPattern matches environment keys that must never be persisted.
// Covers things like RENOS_JWT_SECRET, RENOS_ENCRYPTION_KEY, API keys, DB
// passwords and any AUTH/TOKEN material.
var sensitiveKeyPattern = regexp.MustCompile(`(?i)PASSWORD|SECRET|KEY|TOKEN|AUTH`)

// Options configures a Reporter.
type Options struct {
	App       string
	Version   string
	Commit    string
	BuildTime string
	CrashDir  string
	// LogFile, when set, is included as a log tail inside bundled reports.
	LogFile string
	// MaxLogLines limits the log tail appended to bundles.
	MaxLogLines int
}

// Report is the serialized form of a crash.
type Report struct {
	App        string            `json:"app"`
	Version    string            `json:"version"`
	Commit     string            `json:"commit"`
	BuildTime  string            `json:"build_time"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	GoVersion  string            `json:"go_version"`
	Panic      string            `json:"panic"`
	Stack      string            `json:"stack"`
	Time       time.Time         `json:"time"`
	Goroutines string            `json:"goroutines,omitempty"`
	Env        map[string]string `json:"env"`
}

// Reporter writes crash reports to a crash directory.
type Reporter struct {
	opts Options
	mu   sync.Mutex
}

// New returns a Reporter with defaults applied.
func New(opts Options) *Reporter {
	if opts.CrashDir == "" {
		opts.CrashDir = DefaultDir
	}
	if opts.MaxLogLines <= 0 {
		opts.MaxLogLines = 1000
	}
	return &Reporter{opts: opts}
}

// RedactEnv returns a copy of env with every value whose key matches a
// sensitive pattern replaced by "[REDACTED]". The original map is untouched.
func RedactEnv(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		if sensitiveKeyPattern.MatchString(k) {
			out[k] = "[REDACTED]"
			continue
		}
		out[k] = v
	}
	return out
}

// BuildReport assembles a Report from a recovered panic value and a captured
// stack. The environment snapshot is taken at build time and sanitized.
func (r *Reporter) BuildReport(panicValue interface{}, stack []byte) *Report {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if idx := strings.IndexByte(e, '='); idx > 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}

	goroutines := make([]byte, 1<<20)
	n := runtime.Stack(goroutines, true)
	goroutines = goroutines[:n]

	return &Report{
		App:        r.opts.App,
		Version:    r.opts.Version,
		Commit:     r.opts.Commit,
		BuildTime:  r.opts.BuildTime,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		GoVersion:  runtime.Version(),
		Panic:      fmt.Sprintf("%v", panicValue),
		Stack:      string(stack),
		Time:       time.Now().UTC(),
		Goroutines: string(goroutines),
		Env:        RedactEnv(env),
	}
}

// Write persists a report (and its stack / goroutine dumps) into the crash
// directory and returns the directory holding the report files.
func (r *Reporter) Write(report *Report) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	dir := filepath.Join(r.opts.CrashDir, report.Time.Format("2006-01-02T15-04-05.000"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create crash dir: %w", err)
	}

	if err := writeJSONFile(filepath.Join(dir, "report.json"), report); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "stack.txt"), []byte(report.Stack), 0o644); err != nil {
		return "", err
	}
	if report.Goroutines != "" {
		if err := os.WriteFile(filepath.Join(dir, "goroutines.txt"), []byte(report.Goroutines), 0o644); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// CapturePanic is designed to be used as a deferred statement. It recovers
// from any panic, builds a report and persists it to the crash directory.
// The panic is NOT swallowed — it is re-raised after the report is written.
//
//	func main() {
//	    defer crashreport.New(crashreport.Options{...}).CapturePanic()
//	    ...
//	}
func (r *Reporter) CapturePanic() {
	if rec := recover(); rec != nil {
		r.Capture(rec, runtimeStack())
		panic(rec)
	}
}

// Capture handles an already-recovered panic value (e.g. from middleware).
// It builds, persists and logs the report without re-raising the panic.
func (r *Reporter) Capture(rec interface{}, stack []byte) {
	report := r.BuildReport(rec, stack)
	dir, err := r.Write(report)
	if err != nil {
		slog.Error("crash report: failed to write", "error", err)
		return
	}
	slog.Error("crash report captured",
		"panic", report.Panic,
		"crash_dir", dir,
		"time", report.Time,
	)
}

// runtimeStack returns the stack of the current goroutine.
func runtimeStack() []byte {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, false)
	return buf[:n]
}

// BundleReport creates an in-memory .tar.gz containing report.json, the stack
// trace, the goroutine dump and (optionally) a tail of the log file.
func (r *Reporter) BundleReport(report *Report) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	files := []struct {
		name string
		data []byte
	}{
		{"report.json", mustJSON(report)},
		{"stack.txt", []byte(report.Stack)},
	}
	if report.Goroutines != "" {
		files = append(files, struct {
			name string
			data []byte
		}{"goroutines.txt", []byte(report.Goroutines)})
	}
	if r.opts.LogFile != "" {
		if tail, err := readLogTail(r.opts.LogFile, r.opts.MaxLogLines); err == nil {
			files = append(files, struct {
				name string
				data []byte
			}{"logs.txt", tail})
		}
	}

	for _, f := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name:    f.name,
			Mode:    0o644,
			Size:    int64(len(f.data)),
			ModTime: report.Time,
		}); err != nil {
			return nil, err
		}
		if _, err := tw.Write(f.data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CapturePanic is the package-level convenience for the default reporter.
// Configure the defaults first with Configure.
var defaultReporter = New(Options{})

// Configure sets the package-level default reporter used by CapturePanic.
func Configure(opts Options) {
	defaultReporter = New(opts)
}

// CapturePanic recovers, persists and re-raises a panic using the default
// reporter. Configure the default reporter first with Configure.
func CapturePanic() {
	defaultReporter.CapturePanic()
}

// Capture persists a recovered panic using the default reporter without
// re-raising it. Configure the default reporter first with Configure.
func Capture(rec interface{}, stack []byte) {
	defaultReporter.Capture(rec, stack)
}

func writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func mustJSON(v interface{}) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return []byte(fmt.Sprintf(`{"marshal_error":%q}`, err.Error()))
	}
	return b
}

// readLogTail returns up to maxLines lines from the end of the file at path.
func readLogTail(path string, maxLines int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return bytes.Join(lines, []byte("\n")), nil
}
