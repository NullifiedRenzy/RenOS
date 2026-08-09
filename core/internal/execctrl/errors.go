package execctrl

import (
	"strings"
)

var retryableCodes = map[string]bool{
	"timeout":              true,
	"network_unreachable":  true,
	"connection_refused":   true,
	"agent_unavailable":    true,
	"temporary_failure":    true,
	"device_busy":          true,
	"rate_limited":         true,
	"internal_error":       true,
}

var nonRetryableCodes = map[string]bool{
	"permission_denied":   true,
	"invalid_command":     true,
	"device_not_found":    true,
	"invalid_parameters":  true,
	"unsupported_action":  true,
	"authentication_failed": true,
}

func SetRetryableCodes(codes []string) {
	retryableCodes = make(map[string]bool, len(codes))
	for _, c := range codes {
		retryableCodes[strings.TrimSpace(c)] = true
	}
}

func IsRetryable(errCode string) bool {
	if retryableCodes[errCode] {
		return true
	}
	if nonRetryableCodes[errCode] {
		return false
	}
	return true
}

type ExecError struct {
	Code    string
	Message string
	DeviceID string
}

func (e *ExecError) Error() string {
	return e.Message
}

func NewExecError(code, message, deviceID string) *ExecError {
	return &ExecError{Code: code, Message: message, DeviceID: deviceID}
}

var (
	ErrRateLimited        = &ExecError{Code: "rate_limited", Message: "device rate limited"}
	ErrDeviceUnstable     = &ExecError{Code: "device_unstable", Message: "device marked as unstable"}
	ErrDeviceBusy         = &ExecError{Code: "device_busy", Message: "device busy"}
	ErrTimedOut           = &ExecError{Code: "timeout", Message: "execution timed out"}
	ErrPermissionDenied   = &ExecError{Code: "permission_denied", Message: "permission denied"}
	ErrInvalidCommand     = &ExecError{Code: "invalid_command", Message: "invalid command"}
)
