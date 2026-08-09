using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading;
using System.Threading.Tasks;

namespace RenOS
{
    public class RenOSException : Exception
    {
        public string Code { get; }
        public JsonElement? Details { get; }

        public RenOSException(string code, string message, JsonElement? details = null)
            : base($"[{code}] {message}")
        {
            Code = code;
            Details = details;
        }
    }

    public class PaginatedResponse<T>
    {
        [JsonPropertyName("total")]
        public int Total { get; set; }

        [JsonPropertyName("page")]
        public int Page { get; set; }

        [JsonPropertyName("per_page")]
        public int PerPage { get; set; }

        [JsonPropertyName("data")]
        public List<T> Data { get; set; } = new();
    }

    // --- Model classes ---

    public class Device
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("hostname")] public string Hostname { get; set; } = "";
        [JsonPropertyName("os")] public string Os { get; set; } = "";
        [JsonPropertyName("arch")] public string Arch { get; set; } = "";
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("ip")] public string Ip { get; set; } = "";
        [JsonPropertyName("agent_version")] public string AgentVersion { get; set; } = "";
        [JsonPropertyName("tags")] public Dictionary<string, string> Tags { get; set; } = new();
        [JsonPropertyName("enrolled_at")] public string? EnrolledAt { get; set; }
        [JsonPropertyName("last_seen_at")] public string? LastSeenAt { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("updated_at")] public string? UpdatedAt { get; set; }
    }

    public class DeviceDetail : Device
    {
        [JsonPropertyName("cpu_usage")] public double CpuUsage { get; set; }
        [JsonPropertyName("memory_usage")] public double MemoryUsage { get; set; }
        [JsonPropertyName("disk_usage")] public double DiskUsage { get; set; }
        [JsonPropertyName("load_average")] public double LoadAverage { get; set; }
        [JsonPropertyName("uptime")] public int Uptime { get; set; }
    }

    public class Command
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("device_id")] public string DeviceId { get; set; } = "";
        [JsonPropertyName("command")] public string CommandText { get; set; } = "";
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("exit_code")] public int? ExitCode { get; set; }
        [JsonPropertyName("output")] public string Output { get; set; } = "";
        [JsonPropertyName("error")] public string Error { get; set; } = "";
        [JsonPropertyName("timeout")] public int Timeout { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("completed_at")] public string? CompletedAt { get; set; }
    }

    public class Job
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("type")] public string Type { get; set; } = "";
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("progress")] public double Progress { get; set; }
        [JsonPropertyName("total")] public int Total { get; set; }
        [JsonPropertyName("completed")] public int Completed { get; set; }
        [JsonPropertyName("failed")] public int Failed { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("completed_at")] public string? CompletedAt { get; set; }
        [JsonPropertyName("error")] public string Error { get; set; } = "";
    }

    public class Workflow
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("description")] public string Description { get; set; } = "";
        [JsonPropertyName("steps")] public List<WorkflowStep> Steps { get; set; } = new();
        [JsonPropertyName("enabled")] public bool Enabled { get; set; } = true;
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("updated_at")] public string? UpdatedAt { get; set; }
    }

    public class WorkflowStep
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("workflow_id")] public string WorkflowId { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("type")] public string Type { get; set; } = "";
        [JsonPropertyName("config")] public JsonElement? Config { get; set; }
        [JsonPropertyName("order")] public int Order { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
    }

    public class WorkflowRun
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("workflow_id")] public string WorkflowId { get; set; } = "";
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("started_at")] public string? StartedAt { get; set; }
        [JsonPropertyName("completed_at")] public string? CompletedAt { get; set; }
        [JsonPropertyName("error")] public string Error { get; set; } = "";
        [JsonPropertyName("steps")] public List<WorkflowStep> Steps { get; set; } = new();
    }

    public class User
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("email")] public string Email { get; set; } = "";
        [JsonPropertyName("display_name")] public string DisplayName { get; set; } = "";
        [JsonPropertyName("role")] public string Role { get; set; } = "viewer";
        [JsonPropertyName("avatar_url")] public string AvatarUrl { get; set; } = "";
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("updated_at")] public string? UpdatedAt { get; set; }
    }

    public class Group
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("description")] public string Description { get; set; } = "";
        [JsonPropertyName("device_count")] public int DeviceCount { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("updated_at")] public string? UpdatedAt { get; set; }
    }

    public class Policy
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("description")] public string Description { get; set; } = "";
        [JsonPropertyName("type")] public string Type { get; set; } = "";
        [JsonPropertyName("conditions")] public JsonElement? Conditions { get; set; }
        [JsonPropertyName("action")] public string Action { get; set; } = "";
        [JsonPropertyName("priority")] public int Priority { get; set; }
        [JsonPropertyName("enabled")] public bool Enabled { get; set; } = true;
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("updated_at")] public string? UpdatedAt { get; set; }
    }

    public class Schedule
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("type")] public string Type { get; set; } = "";
        [JsonPropertyName("cron_expression")] public string CronExpression { get; set; } = "";
        [JsonPropertyName("interval_seconds")] public int IntervalSeconds { get; set; }
        [JsonPropertyName("actions")] public List<JsonElement> Actions { get; set; } = new();
        [JsonPropertyName("enabled")] public bool Enabled { get; set; } = true;
        [JsonPropertyName("last_run")] public string? LastRun { get; set; }
        [JsonPropertyName("next_run")] public string? NextRun { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("updated_at")] public string? UpdatedAt { get; set; }
    }

    public class VaultSecret
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("path")] public string Path { get; set; } = "";
        [JsonPropertyName("secret_type")] public string SecretType { get; set; } = "";
        [JsonPropertyName("version")] public int Version { get; set; } = 1;
        [JsonPropertyName("metadata")] public JsonElement? Metadata { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("updated_at")] public string? UpdatedAt { get; set; }
    }

    public class AlertInstance
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("rule_id")] public string RuleId { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("severity")] public string Severity { get; set; } = "";
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("metric_value")] public double MetricValue { get; set; }
        [JsonPropertyName("starts_at")] public string? StartsAt { get; set; }
        [JsonPropertyName("acknowledged_at")] public string? AcknowledgedAt { get; set; }
        [JsonPropertyName("resolved_at")] public string? ResolvedAt { get; set; }
    }

    public class AlertRule
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("metric")] public string Metric { get; set; } = "";
        [JsonPropertyName("condition")] public string Condition { get; set; } = "";
        [JsonPropertyName("threshold")] public double Threshold { get; set; }
        [JsonPropertyName("severity")] public string Severity { get; set; } = "";
        [JsonPropertyName("enabled")] public bool Enabled { get; set; } = true;
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
    }

    public class ClusterNode
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("name")] public string Name { get; set; } = "";
        [JsonPropertyName("address")] public string Address { get; set; } = "";
        [JsonPropertyName("role")] public string Role { get; set; } = "";
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("joined_at")] public string? JoinedAt { get; set; }
        [JsonPropertyName("last_heartbeat")] public string? LastHeartbeat { get; set; }
    }

    public class TerminalSession
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("device_id")] public string DeviceId { get; set; } = "";
        [JsonPropertyName("user_id")] public string UserId { get; set; } = "";
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("closed_at")] public string? ClosedAt { get; set; }
    }

    public class Execution
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("device_id")] public string DeviceId { get; set; } = "";
        [JsonPropertyName("command")] public string Command { get; set; } = "";
        [JsonPropertyName("args")] public List<string> Args { get; set; } = new();
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("exit_code")] public int? ExitCode { get; set; }
        [JsonPropertyName("output")] public string Output { get; set; } = "";
        [JsonPropertyName("error")] public string Error { get; set; } = "";
        [JsonPropertyName("workdir")] public string Workdir { get; set; } = "";
        [JsonPropertyName("pid")] public int Pid { get; set; }
        [JsonPropertyName("started_at")] public string? StartedAt { get; set; }
        [JsonPropertyName("completed_at")] public string? CompletedAt { get; set; }
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
    }

    public class FileTransfer
    {
        [JsonPropertyName("id")] public string Id { get; set; } = "";
        [JsonPropertyName("device_id")] public string DeviceId { get; set; } = "";
        [JsonPropertyName("filename")] public string Filename { get; set; } = "";
        [JsonPropertyName("path")] public string Path { get; set; } = "";
        [JsonPropertyName("direction")] public string Direction { get; set; } = "";
        [JsonPropertyName("size")] public int Size { get; set; }
        [JsonPropertyName("transferred")] public int Transferred { get; set; }
        [JsonPropertyName("status")] public string Status { get; set; } = "";
        [JsonPropertyName("progress")] public double Progress { get; set; }
        [JsonPropertyName("error")] public string Error { get; set; } = "";
        [JsonPropertyName("created_at")] public string? CreatedAt { get; set; }
        [JsonPropertyName("completed_at")] public string? CompletedAt { get; set; }
    }

    // --- Client ---

    public class RenOSClient : IDisposable
    {
        private readonly HttpClient _http;
        private readonly string _baseUrl;
        private string? _token;
        private string? _apiKey;

        private static readonly JsonSerializerOptions JsonOptions = new()
        {
            PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        };

        public RenOSClient(string baseUrl = "http://localhost:8443", string? token = null, string? apiKey = null)
        {
            _baseUrl = baseUrl.TrimEnd('/');
            _token = token;
            _apiKey = apiKey;
            _http = new HttpClient();
            _http.DefaultRequestHeaders.Accept.Add(
                new MediaTypeWithQualityHeaderValue("application/json"));
        }

        public void Dispose() => _http.Dispose();

        public void SetToken(string token) => _token = token;
        public void SetApiKey(string key) => _apiKey = key;

        public static async Task<RenOSClient> LoginAsync(string serverUrl, string email, string password)
        {
            var baseUrl = (serverUrl ?? "http://localhost:8443").TrimEnd('/');
            using var client = new HttpClient();
            var payload = JsonSerializer.Serialize(new { email, password }, JsonOptions);
            var resp = await client.PostAsync(
                $"{baseUrl}/api/v1/auth/login",
                new StringContent(payload, Encoding.UTF8, "application/json"));

            if (!resp.IsSuccessStatusCode)
            {
                var body = await resp.Content.ReadAsStringAsync();
                throw new RenOSException("LOGIN_FAILED", $"Login failed ({(int)resp.StatusCode}): {body}");
            }

            var json = await resp.Content.ReadAsStringAsync();
            var data = JsonSerializer.Deserialize<LoginResponse>(json, JsonOptions)
                ?? throw new RenOSException("PARSE_ERROR", "Failed to parse login response");
            return new RenOSClient(baseUrl, data.Token);
        }

        private async Task<T> RequestAsync<T>(string method, string path, object? body = null,
            Dictionary<string, string>? queryParams = null)
        {
            var url = $"{_baseUrl}{path}";
            if (queryParams != null && queryParams.Count > 0)
            {
                var qs = string.Join("&",
                    queryParams.Select(kv => $"{Uri.EscapeDataString(kv.Key)}={Uri.EscapeDataString(kv.Value)}"));
                url += "?" + qs;
            }

            var request = new HttpRequestMessage(new HttpMethod(method), url);
            request.Headers.Add("Content-Type", "application/json");
            if (!string.IsNullOrEmpty(_token))
                request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", _token);
            if (!string.IsNullOrEmpty(_apiKey))
                request.Headers.Add("X-API-Key", _apiKey);

            if (body != null)
            {
                request.Content = new StringContent(
                    JsonSerializer.Serialize(body, JsonOptions),
                    Encoding.UTF8,
                    "application/json");
            }

            var resp = await _http.SendAsync(request);
            var respBody = await resp.Content.ReadAsStringAsync();

            if (!resp.IsSuccessStatusCode)
            {
                string code = "API_ERROR";
                string message = $"HTTP {(int)resp.StatusCode}";
                JsonElement? details = null;
                try
                {
                    var err = JsonSerializer.Deserialize<JsonElement>(respBody);
                    if (err.TryGetProperty("code", out var c)) code = c.GetString() ?? code;
                    if (err.TryGetProperty("message", out var m)) message = m.GetString() ?? message;
                    if (err.TryGetProperty("details", out var d)) details = d;
                }
                catch { message = respBody; }
                throw new RenOSException(code, message, details);
            }

            if (typeof(T) == typeof(Empty)) return (T)(object)new Empty();
            return JsonSerializer.Deserialize<T>(respBody, JsonOptions)
                ?? throw new RenOSException("PARSE_ERROR", "Failed to parse response");
        }

        private async Task<PaginatedResponse<T>> PaginatedAsync<T>(string path, int page = 1, int perPage = 50)
        {
            var result = await RequestAsync<PaginatedResponse<T>>("GET", path,
                queryParams: new() { ["page"] = page.ToString(), ["per_page"] = perPage.ToString() });
            return result;
        }

        // --- API methods ---

        public Task<PaginatedResponse<Device>> ListDevices(int page = 1, int perPage = 50)
            => PaginatedAsync<Device>("/api/v1/devices", page, perPage);

        public Task<Device> GetDevice(string id)
            => RequestAsync<Device>("GET", $"/api/v1/devices/{id}");

        public Task<Command> ExecuteCommand(string deviceId, string command)
            => RequestAsync<Command>("POST", $"/api/v1/devices/{deviceId}/commands",
                new { command });

        public Task<Command> GetCommand(string deviceId, string cmdId)
            => RequestAsync<Command>("GET", $"/api/v1/devices/{deviceId}/commands/{cmdId}");

        public Task<PaginatedResponse<Command>> ListCommands(string deviceId, int page = 1, int perPage = 50)
            => PaginatedAsync<Command>($"/api/v1/devices/{deviceId}/commands", page, perPage);

        public Task<PaginatedResponse<Job>> ListJobs(int page = 1, int perPage = 50)
            => PaginatedAsync<Job>("/api/v1/jobs", page, perPage);

        public Task<Job> GetJob(string id)
            => RequestAsync<Job>("GET", $"/api/v1/jobs/{id}");

        public Task CancelJob(string id)
            => RequestAsync<Empty>("POST", $"/api/v1/jobs/{id}/cancel");

        public Task RetryJob(string id)
            => RequestAsync<Empty>("POST", $"/api/v1/jobs/{id}/retry");

        public Task<PaginatedResponse<Workflow>> ListWorkflows(int page = 1, int perPage = 50)
            => PaginatedAsync<Workflow>("/api/v1/workflows", page, perPage);

        public Task<Workflow> GetWorkflow(string id)
            => RequestAsync<Workflow>("GET", $"/api/v1/workflows/{id}");

        public Task<WorkflowRun> ExecuteWorkflow(string id)
            => RequestAsync<WorkflowRun>("POST", $"/api/v1/workflows/{id}/execute");

        public Task<PaginatedResponse<User>> ListUsers(int page = 1, int perPage = 50)
            => PaginatedAsync<User>("/api/v1/users", page, perPage);

        public Task<PaginatedResponse<AlertInstance>> ListAlerts(int page = 1, int perPage = 50)
            => PaginatedAsync<AlertInstance>("/api/v1/alerts/active", page, perPage);

        public Task<PaginatedResponse<Group>> ListGroups(int page = 1, int perPage = 50)
            => PaginatedAsync<Group>("/api/v1/groups", page, perPage);

        public Task<Group> GetGroup(string id)
            => RequestAsync<Group>("GET", $"/api/v1/groups/{id}");

        public Task<PaginatedResponse<Schedule>> ListSchedules(int page = 1, int perPage = 50)
            => PaginatedAsync<Schedule>("/api/v1/schedules", page, perPage);

        public Task<PaginatedResponse<Policy>> ListPolicies(int page = 1, int perPage = 50)
            => PaginatedAsync<Policy>("/api/v1/policies", page, perPage);

        public Task<PaginatedResponse<VaultSecret>> ListVaultSecrets(int page = 1, int perPage = 50)
            => PaginatedAsync<VaultSecret>("/api/v1/vault/secrets", page, perPage);

        public Task<PaginatedResponse<ClusterNode>> ListClusterNodes(int page = 1, int perPage = 50)
            => PaginatedAsync<ClusterNode>("/api/v1/cluster/nodes", page, perPage);

        public Task<PaginatedResponse<TerminalSession>> ListTerminalSessions(string deviceId, int page = 1, int perPage = 50)
            => PaginatedAsync<TerminalSession>($"/api/v1/devices/{deviceId}/terminal/sessions", page, perPage);

        public Task<PaginatedResponse<Execution>> ListExecutions(string deviceId, int page = 1, int perPage = 50)
            => PaginatedAsync<Execution>($"/api/v1/devices/{deviceId}/executions", page, perPage);

        public Task<Execution> CreateExecution(string deviceId, string command, List<string>? args = null)
            => RequestAsync<Execution>("POST", $"/api/v1/devices/{deviceId}/executions",
                new { command, args = args ?? new List<string>() });

        public Task<Execution> GetExecution(string deviceId, string execId)
            => RequestAsync<Execution>("GET", $"/api/v1/devices/{deviceId}/executions/{execId}");

        public Task CancelExecution(string deviceId, string execId)
            => RequestAsync<Empty>("POST", $"/api/v1/devices/{deviceId}/executions/{execId}/cancel");

        // --- Streaming ---

        public async Task<Stream> StreamEvents(string path)
        {
            var url = $"{_baseUrl}{path}";
            var request = new HttpRequestMessage(HttpMethod.Get, url);
            request.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("text/event-stream"));
            if (!string.IsNullOrEmpty(_token))
                request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", _token);
            if (!string.IsNullOrEmpty(_apiKey))
                request.Headers.Add("X-API-Key", _apiKey);

            var resp = await _http.SendAsync(request, HttpCompletionOption.ResponseHeadersRead);
            if (!resp.IsSuccessStatusCode)
                throw new RenOSException("STREAM_ERROR", $"Stream returned {(int)resp.StatusCode}");

            var stream = new Stream();
            _ = Task.Run(() => stream.ReadAsync(resp.Content));
            return stream;
        }

        // --- WebSocket ---

        public RenOSWebSocket ConnectWebSocket(string path)
        {
            var wsUrl = _baseUrl.Replace("https://", "wss://").Replace("http://", "ws://") + path;
            var ws = new RenOSWebSocket(wsUrl, _token, _apiKey);
            return ws;
        }

        // --- Helper types ---

        private class Empty { }

        private class LoginResponse
        {
            [JsonPropertyName("token")]
            public string Token { get; set; } = "";
        }
    }

    // --- Stream ---

    public class Stream
    {
        private readonly List<Action<string>> _listeners = new();
        private bool _closed;

        public event Action<string>? OnData;

        internal async Task ReadAsync(HttpContent content)
        {
            using var stream = await content.ReadAsStreamAsync();
            using var reader = new StreamReader(stream);
            while (!_closed)
            {
                var line = await reader.ReadLineAsync();
                if (line == null) break;
                if (line.StartsWith("data: "))
                {
                    var data = line[6..];
                    OnData?.Invoke(data);
                    lock (_listeners)
                    {
                        foreach (var listener in _listeners)
                            listener(data);
                    }
                }
            }
        }

        public IDisposable Subscribe(Action<string> handler)
        {
            lock (_listeners) _listeners.Add(handler);
            return new Subscription(() => { lock (_listeners) _listeners.Remove(handler); });
        }

        public void Close() => _closed = true;

        private class Subscription : IDisposable
        {
            private readonly Action _dispose;
            public Subscription(Action dispose) => _dispose = dispose;
            public void Dispose() => _dispose();
        }
    }

    // --- WebSocket ---

    public class RenOSWebSocket : IDisposable
    {
        private ClientWebSocket? _ws;
        private readonly Uri _uri;
        private readonly string? _token;
        private readonly string? _apiKey;
        private CancellationTokenSource _cts = new();
        private bool _reconnectEnabled;
        private int _reconnectDelay = 1000;
        private const int MaxReconnectDelay = 30000;
        private bool _closed;

        public event Action<string>? OnMessage;

        public RenOSWebSocket(string url, string? token, string? apiKey)
        {
            _uri = new Uri(url);
            _token = token;
            _apiKey = apiKey;
            ConnectAsync().ConfigureAwait(false);
        }

        private async Task ConnectAsync()
        {
            if (_closed) return;
            try
            {
                _ws?.Dispose();
                _ws = new ClientWebSocket();
                if (!string.IsNullOrEmpty(_token))
                    _ws.Options.SetRequestHeader("Authorization", $"Bearer {_token}");
                if (!string.IsNullOrEmpty(_apiKey))
                    _ws.Options.SetRequestHeader("X-API-Key", _apiKey);

                await _ws.ConnectAsync(_uri, _cts.Token);
                _reconnectDelay = 1000;
                _ = ReceiveLoopAsync();
            }
            catch
            {
                if (_reconnectEnabled && !_closed)
                {
                    await Task.Delay(_reconnectDelay);
                    _reconnectDelay = Math.Min(_reconnectDelay * 2, MaxReconnectDelay);
                    _ = ConnectAsync();
                }
            }
        }

        private async Task ReceiveLoopAsync()
        {
            var buffer = new byte[4096];
            try
            {
                while (_ws?.State == WebSocketState.Open && !_closed)
                {
                    var result = await _ws.ReceiveAsync(new ArraySegment<byte>(buffer), _cts.Token);
                    if (result.MessageType == WebSocketMessageType.Close)
                        break;

                    var text = Encoding.UTF8.GetString(buffer, 0, result.Count);
                    OnMessage?.Invoke(text);
                }
            }
            catch { }

            if (_reconnectEnabled && !_closed)
            {
                await Task.Delay(_reconnectDelay);
                _reconnectDelay = Math.Min(_reconnectDelay * 2, MaxReconnectDelay);
                _ = ConnectAsync();
            }
        }

        public void EnableReconnect() => _reconnectEnabled = true;
        public void DisableReconnect() => _reconnectEnabled = false;

        public async Task SendAsync(string message)
        {
            if (_ws?.State == WebSocketState.Open)
            {
                var bytes = Encoding.UTF8.GetBytes(message);
                await _ws.SendAsync(new ArraySegment<byte>(bytes),
                    WebSocketMessageType.Text, true, _cts.Token);
            }
        }

        public void Dispose()
        {
            _closed = true;
            _cts.Cancel();
            _ws?.Dispose();
            _ws = null;
        }
    }
}
