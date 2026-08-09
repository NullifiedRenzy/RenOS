package renossdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	DefaultBaseURL = "http://localhost:8443"
	DefaultTimeout = 30 * time.Second
)

type Config struct {
	BaseURL    string
	Token      string
	APIKey     string
	HTTPClient *http.Client
	Timeout    time.Duration
}

type Client struct {
	baseURL    string
	token      string
	apiKey     string
	httpClient *http.Client
	mu         sync.RWMutex
}

func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func NewClientFromConfig(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{
		baseURL:    cfg.BaseURL,
		token:      cfg.Token,
		apiKey:     cfg.APIKey,
		httpClient: cfg.HTTPClient,
	}
}

type LoginResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
}

func NewClientFromLogin(serverURL, email, password string) (*Client, error) {
	if serverURL == "" {
		serverURL = DefaultBaseURL
	}
	body := map[string]string{"email": email, "password": password}
	data, _ := json.Marshal(body)
	resp, err := http.Post(serverURL+"/api/v1/auth/login", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	var lr LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}
	return NewClient(serverURL, lr.Token), nil
}

func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

func (c *Client) SetAPIKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKey = key
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (c *Client) Request(method, path string, body, result any) error {
	c.mu.RLock()
	token := c.token
	apiKey := c.apiKey
	c.mu.RUnlock()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("request marshal error: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("unexpected status %d", resp.StatusCode)
		}
		return &apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("response decode error: %w", err)
		}
	}
	return nil
}

type PaginatedList[T any] struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Items   []T `json:"data,omitempty"`
}

func (c *Client) ListDevices(page, perPage int) (*PaginatedList[Device], error) {
	result := &PaginatedList[Device]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/devices?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) GetDevice(id string) (*Device, error) {
	result := &Device{}
	err := c.Request("GET", "/api/v1/devices/"+id, nil, result)
	return result, err
}

func (c *Client) ExecuteCommand(deviceID, command string) (*Command, error) {
	result := &Command{}
	body := map[string]string{"command": command}
	err := c.Request("POST", "/api/v1/devices/"+deviceID+"/commands", body, result)
	return result, err
}

func (c *Client) GetCommand(deviceID, cmdID string) (*Command, error) {
	result := &Command{}
	err := c.Request("GET", "/api/v1/devices/"+deviceID+"/commands/"+cmdID, nil, result)
	return result, err
}

func (c *Client) ListCommands(deviceID string, page, perPage int) (*PaginatedList[Command], error) {
	result := &PaginatedList[Command]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/devices/%s/commands?page=%d&per_page=%d", deviceID, page, perPage), nil, result)
	return result, err
}

func (c *Client) ListJobs(page, perPage int) (*PaginatedList[Job], error) {
	result := &PaginatedList[Job]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/jobs?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) GetJob(id string) (*Job, error) {
	result := &Job{}
	err := c.Request("GET", "/api/v1/jobs/"+id, nil, result)
	return result, err
}

func (c *Client) CancelJob(id string) error {
	return c.Request("POST", "/api/v1/jobs/"+id+"/cancel", nil, nil)
}

func (c *Client) RetryJob(id string) error {
	return c.Request("POST", "/api/v1/jobs/"+id+"/retry", nil, nil)
}

func (c *Client) ListWorkflows(page, perPage int) (*PaginatedList[Workflow], error) {
	result := &PaginatedList[Workflow]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/workflows?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) GetWorkflow(id string) (*Workflow, error) {
	result := &Workflow{}
	err := c.Request("GET", "/api/v1/workflows/"+id, nil, result)
	return result, err
}

func (c *Client) ExecuteWorkflow(id string) (*WorkflowRun, error) {
	result := &WorkflowRun{}
	err := c.Request("POST", "/api/v1/workflows/"+id+"/execute", nil, result)
	return result, err
}

func (c *Client) ListUsers(page, perPage int) (*PaginatedList[User], error) {
	result := &PaginatedList[User]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/users?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) ListAlerts(page, perPage int) (*PaginatedList[AlertInstance], error) {
	result := &PaginatedList[AlertInstance]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/alerts/active?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) ListGroups(page, perPage int) (*PaginatedList[Group], error) {
	result := &PaginatedList[Group]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/groups?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) GetGroup(id string) (*Group, error) {
	result := &Group{}
	err := c.Request("GET", "/api/v1/groups/"+id, nil, result)
	return result, err
}

func (c *Client) ListSchedules(page, perPage int) (*PaginatedList[Schedule], error) {
	result := &PaginatedList[Schedule]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/schedules?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) ListPolicies(page, perPage int) (*PaginatedList[Policy], error) {
	result := &PaginatedList[Policy]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/policies?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) ListVaultSecrets(page, perPage int) (*PaginatedList[VaultSecret], error) {
	result := &PaginatedList[VaultSecret]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/vault/secrets?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) ListClusterNodes(page, perPage int) (*PaginatedList[ClusterNode], error) {
	result := &PaginatedList[ClusterNode]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/cluster/nodes?page=%d&per_page=%d", page, perPage), nil, result)
	return result, err
}

func (c *Client) ListTerminalSessions(deviceID string, page, perPage int) (*PaginatedList[TerminalSession], error) {
	result := &PaginatedList[TerminalSession]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/devices/%s/terminal/sessions?page=%d&per_page=%d", deviceID, page, perPage), nil, result)
	return result, err
}

func (c *Client) ListExecutions(deviceID string, page, perPage int) (*PaginatedList[Execution], error) {
	result := &PaginatedList[Execution]{}
	err := c.Request("GET", fmt.Sprintf("/api/v1/devices/%s/executions?page=%d&per_page=%d", deviceID, page, perPage), nil, result)
	return result, err
}

func (c *Client) CreateExecution(deviceID, command string, args []string) (*Execution, error) {
	result := &Execution{}
	body := map[string]any{"command": command, "args": args}
	err := c.Request("POST", "/api/v1/devices/"+deviceID+"/executions", body, result)
	return result, err
}

func (c *Client) GetExecution(deviceID, execID string) (*Execution, error) {
	result := &Execution{}
	err := c.Request("GET", "/api/v1/devices/"+deviceID+"/executions/"+execID, nil, result)
	return result, err
}

func (c *Client) CancelExecution(deviceID, execID string) error {
	return c.Request("POST", "/api/v1/devices/"+deviceID+"/executions/"+execID+"/cancel", nil, nil)
}

type Stream struct {
	events    chan StreamEvent
	done      chan struct{}
	closeOnce sync.Once
}

type StreamEvent struct {
	Type string
	Data json.RawMessage
	Err  error
}

func (c *Client) Stream(path string) (*Stream, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("stream request error: %w", err)
	}
	c.mu.RLock()
	token := c.token
	apiKey := c.apiKey
	c.mu.RUnlock()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stream connection error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("stream returned status %d", resp.StatusCode)
	}

	s := &Stream{
		events: make(chan StreamEvent, 64),
		done:   make(chan struct{}),
	}

	go func() {
		defer resp.Body.Close()
		defer close(s.events)
		dec := json.NewDecoder(resp.Body)
		for {
			select {
			case <-s.done:
				return
			default:
			}
			var event StreamEvent
			if err := dec.Decode(&event); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				select {
				case s.events <- StreamEvent{Err: err}:
				case <-s.done:
				}
				return
			}
			select {
			case s.events <- event:
			case <-s.done:
				return
			}
		}
	}()

	return s, nil
}

func (s *Stream) Events() <-chan StreamEvent {
	return s.events
}

func (s *Stream) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
}

type WebSocket struct {
	conn        *websocket.Conn
	url         string
	token       string
	apiKey      string
	messages    chan []byte
	done        chan struct{}
	closeOnce   sync.Once
	mu          sync.Mutex
	reconnect   bool
	reconnectFn func() (string, error)
}

func (c *Client) WebSocket(path string) (*WebSocket, error) {
	c.mu.RLock()
	token := c.token
	apiKey := c.apiKey
	c.mu.RUnlock()

	wsURL := c.baseURL + path
	wsURL = wsURLReplaceScheme(wsURL)

	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	if apiKey != "" {
		header.Set("X-API-Key", apiKey)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("websocket dial error: %w", err)
	}

	ws := &WebSocket{
		conn:     conn,
		url:      wsURL,
		token:    token,
		apiKey:   apiKey,
		messages: make(chan []byte, 64),
		done:     make(chan struct{}),
	}
	go ws.readLoop()
	return ws, nil
}

func wsURLReplaceScheme(u string) string {
	if len(u) > 4 && u[:4] == "http" {
		return "ws" + u[4:]
	}
	return u
}

func (ws *WebSocket) readLoop() {
	defer close(ws.messages)
	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			if ws.reconnect && ws.reconnectFn != nil {
				ws.mu.Lock()
				ws.conn.Close()
				newURL, err := ws.reconnectFn()
				if err != nil {
					ws.mu.Unlock()
					return
				}
				if newURL != "" {
					ws.url = newURL
				}
				header := http.Header{}
				if ws.token != "" {
					header.Set("Authorization", "Bearer "+ws.token)
				}
				if ws.apiKey != "" {
					header.Set("X-API-Key", ws.apiKey)
				}
				conn, _, dialErr := websocket.DefaultDialer.Dial(ws.url, header)
				if dialErr != nil {
					ws.mu.Unlock()
					return
				}
				ws.conn = conn
				ws.mu.Unlock()
				continue
			}
			return
		}
		select {
		case ws.messages <- message:
		case <-ws.done:
			return
		}
	}
}

func (ws *WebSocket) Messages() <-chan []byte {
	return ws.messages
}

func (ws *WebSocket) Send(message []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.WriteMessage(websocket.TextMessage, message)
}

func (ws *WebSocket) EnableReconnect(fn func() (string, error)) {
	ws.reconnect = true
	ws.reconnectFn = fn
}

func (ws *WebSocket) Close() {
	ws.closeOnce.Do(func() {
		close(ws.done)
		ws.mu.Lock()
		ws.conn.Close()
		ws.mu.Unlock()
	})
}

// Model types

type Device struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Hostname     string            `json:"hostname"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	Status       string            `json:"status"`
	IP           string            `json:"ip"`
	AgentVersion string            `json:"agent_version"`
	Tags         map[string]string `json:"tags"`
	EnrolledAt   string            `json:"enrolled_at"`
	LastSeenAt   string            `json:"last_seen_at"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

type DeviceDetail struct {
	Device
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	LoadAverage float64 `json:"load_average"`
	Uptime      int     `json:"uptime"`
}

type Command struct {
	ID          string `json:"id"`
	DeviceID    string `json:"device_id"`
	Command     string `json:"command"`
	Status      string `json:"status"`
	ExitCode    int    `json:"exit_code"`
	Output      string `json:"output"`
	Error       string `json:"error"`
	Timeout     int    `json:"timeout"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at"`
}

type Job struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	Total       int     `json:"total"`
	Completed   int     `json:"completed"`
	Failed      int     `json:"failed"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt string  `json:"completed_at"`
	Error       string  `json:"error"`
}

type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Steps       []WorkflowStep `json:"steps"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

type WorkflowStep struct {
	ID         string `json:"id"`
	WorkflowID string `json:"workflow_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Config     any    `json:"config"`
	Order      int    `json:"order"`
	CreatedAt  string `json:"created_at"`
}

type WorkflowRun struct {
	ID          string          `json:"id"`
	WorkflowID  string          `json:"workflow_id"`
	Status      string          `json:"status"`
	StartedAt   string          `json:"started_at"`
	CompletedAt string          `json:"completed_at"`
	Error       string          `json:"error"`
	Steps       []WorkflowStep `json:"steps"`
}

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	AvatarURL   string `json:"avatar_url"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DeviceCount int    `json:"device_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Policy struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Conditions  any    `json:"conditions"`
	Action      string `json:"action"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Schedule struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	CronExpression  string `json:"cron_expression"`
	IntervalSeconds int    `json:"interval_seconds"`
	Actions         []any  `json:"actions"`
	Enabled         bool   `json:"enabled"`
	LastRun         string `json:"last_run"`
	NextRun         string `json:"next_run"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type VaultSecret struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	SecretType string `json:"secret_type"`
	Version    int    `json:"version"`
	Metadata   any    `json:"metadata"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type AlertInstance struct {
	ID             string  `json:"id"`
	RuleID         string  `json:"rule_id"`
	Name           string  `json:"name"`
	Severity       string  `json:"severity"`
	Status         string  `json:"status"`
	MetricValue    float64 `json:"metric_value"`
	StartsAt       string  `json:"starts_at"`
	AcknowledgedAt string  `json:"acknowledged_at"`
	ResolvedAt     string  `json:"resolved_at"`
}

type AlertRule struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Condition string  `json:"condition"`
	Threshold float64 `json:"threshold"`
	Severity  string  `json:"severity"`
	Enabled   bool    `json:"enabled"`
	CreatedAt string  `json:"created_at"`
}

type ClusterNode struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	JoinedAt      string `json:"joined_at"`
	LastHeartbeat string `json:"last_heartbeat"`
}

type TerminalSession struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	ClosedAt  string `json:"closed_at"`
}

type Execution struct {
	ID          string   `json:"id"`
	DeviceID    string   `json:"device_id"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Status      string   `json:"status"`
	ExitCode    int      `json:"exit_code"`
	Output      string   `json:"output"`
	Error       string   `json:"error"`
	Workdir     string   `json:"workdir"`
	PID         int      `json:"pid"`
	StartedAt   string   `json:"started_at"`
	CompletedAt string   `json:"completed_at"`
	CreatedAt   string   `json:"created_at"`
}

type FileTransfer struct {
	ID          string  `json:"id"`
	DeviceID    string  `json:"device_id"`
	Filename    string  `json:"filename"`
	Path        string  `json:"path"`
	Direction   string  `json:"direction"`
	Size        int     `json:"size"`
	Transferred int     `json:"transferred"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	Error       string  `json:"error"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt string  `json:"completed_at"`
}
