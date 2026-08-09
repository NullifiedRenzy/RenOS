package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const configDir = ".renos"
const configFile = "credentials.json"

type credentials struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
}

func credPath() string {
	home, _ := os.UserHomeDir()
	return home + "/" + configDir + "/" + configFile
}

func loadCreds() *credentials {
	data, err := os.ReadFile(credPath())
	if err != nil {
		return nil
	}
	var c credentials
	json.Unmarshal(data, &c)
	return &c
}

func saveCreds(c *credentials) {
	home, _ := os.UserHomeDir()
	os.MkdirAll(home+"/"+configDir, 0700)
	data, _ := json.Marshal(c)
	os.WriteFile(credPath(), data, 0600)
}

func apiRequest(method, path string, body interface{}) (*http.Response, error) {
	creds := loadCreds()
	if creds == nil {
		fmt.Fprintln(os.Stderr, "not logged in. Run 'renctl login' first.")
		os.Exit(1)
	}
	url := strings.TrimRight(creds.ServerURL, "/") + "/api/v1" + path
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, url, strings.NewReader(string(b)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	req.Header.Set("Authorization", "Bearer "+creds.Token)
	return http.DefaultClient.Do(req)
}

func printJSON(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func Login(args []string) {
	serverURL := ""
	email := ""
	password := ""
	for i, a := range args {
		switch a {
		case "--server", "-s":
			if i+1 < len(args) {
				serverURL = args[i+1]
			}
		case "--email", "-e":
			if i+1 < len(args) {
				email = args[i+1]
			}
		case "--password", "-p":
			if i+1 < len(args) {
				password = args[i+1]
			}
		case "--help", "-h":
			fmt.Println(`Usage: renctl login [options]
Options:
  --server, -s <url>    RenOS server URL
  --email, -e <email>   Login email
  --password, -p <pass> Login password (omit for prompt)`)
			return
		}
	}

	if serverURL == "" {
		fmt.Print("Server URL: ")
		fmt.Scanln(&serverURL)
	}
	if email == "" {
		fmt.Print("Email: ")
		fmt.Scanln(&email)
	}
	if password == "" {
		fmt.Print("Password: ")
		fmt.Scanln(&password)
	}

	resp, err := http.Post(strings.TrimRight(serverURL, "/")+"/api/v1/auth/login",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse response: %v\n", err)
		os.Exit(1)
	}

	saveCreds(&credentials{ServerURL: serverURL, Token: result.Token})
	fmt.Printf("logged in as %s (%s)\n", result.User.Email, result.User.Role)
}

func Logout() {
	os.Remove(credPath())
	fmt.Println("logged out")
}

func Devices(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl devices <subcommand> [options]
Subcommands:
  list                    List all devices
  get <id>                Get device details
  tag <id> <key>=<value>  Tag a device
  exec <id> <command>     Execute command on device
  logs <id>               View device logs`)
		return
	}

	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/devices", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl devices get <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("GET", "/devices/"+args[1], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "exec":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: renctl devices exec <id> <command>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/devices/"+args[1]+"/commands", map[string]string{
			"command": strings.Join(args[2:], " "),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "tag":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: renctl devices tag <id> <key>=<value>")
			os.Exit(1)
		}
		kv := strings.SplitN(args[2], "=", 2)
		if len(kv) != 2 {
			fmt.Fprintln(os.Stderr, "invalid tag format. use key=value")
			os.Exit(1)
		}
		resp, err := apiRequest("PUT", "/tags/devices/"+args[1], map[string]string{
			"key":   kv[0],
			"value": kv[1],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Jobs(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl jobs <subcommand> [options]
Subcommands:
  list             List all jobs
  get <id>         Get job details
  cancel <id>      Cancel a job
  retry <id>       Retry a failed job
  stats            Show job statistics`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/jobs", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "stats":
		resp, err := apiRequest("GET", "/jobs/stats", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl jobs get <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("GET", "/jobs/"+args[1], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "cancel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl jobs cancel <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/jobs/"+args[1]+"/cancel", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "retry":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl jobs retry <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/jobs/"+args[1]+"/retry", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Workflows(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl workflows <subcommand> [options]
Subcommands:
  list             List all workflows
  get <id>         Get workflow details
  create <file>    Create workflow from JSON file
  run <id>         Execute a workflow
  toggle <id>      Enable/disable a workflow
  delete <id>      Delete a workflow`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/workflows", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl workflows get <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("GET", "/workflows/"+args[1], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl workflows run <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/workflows/"+args[1]+"/execute", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "toggle":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl workflows toggle <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/workflows/"+args[1]+"/toggle", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "create":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl workflows create <file.json>")
			os.Exit(1)
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
			os.Exit(1)
		}
		var body map[string]interface{}
		json.Unmarshal(data, &body)
		resp, err := apiRequest("POST", "/workflows", body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Plugins(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl plugins <subcommand> [options]
Subcommands:
  list                List installed plugins
  get <name>          Get plugin details
  install <name>      Install a plugin from marketplace
  remove <name>       Remove a plugin
  enable <name>       Enable a plugin
  disable <name>      Disable a plugin
  marketplace         List marketplace plugins`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/plugin-platform", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "marketplace":
		resp, err := apiRequest("GET", "/marketplace", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "install":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl plugins install <name>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/marketplace/"+args[1]+"/install", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl plugins remove <name>")
			os.Exit(1)
		}
		resp, err := apiRequest("DELETE", "/plugin-platform/"+args[1], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "enable":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl plugins enable <name>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/plugin-platform/"+args[1]+"/enable", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "disable":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl plugins disable <name>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/plugin-platform/"+args[1]+"/disable", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Cluster(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl cluster <subcommand> [options]
Subcommands:
  dashboard    Show cluster dashboard
  nodes        List cluster nodes
  tasks        List cluster tasks
  state        Show cluster state
  elections    List cluster elections`)
		return
	}
	path := "/cluster/" + args[0]
	resp, err := apiRequest("GET", path, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	printJSON(result)
}

func Vault(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl vault <subcommand> [options]
Subcommands:
  list               List all secrets
  get <id>           Get secret details
  create <name>      Create a new secret
  delete <id>        Delete a secret
  keys               List encryption keys`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/vault/secrets", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "keys":
		resp, err := apiRequest("GET", "/vault/keys", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl vault get <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("GET", "/vault/secrets/"+args[1], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Users(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl users <subcommand> [options]
Subcommands:
  list                 List all users
  update-role <id> <role>  Update user role (admin/operator/viewer)`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/users", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "update-role":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: renctl users update-role <id> <role>")
			os.Exit(1)
		}
		resp, err := apiRequest("PUT", "/users/"+args[1]+"/role", map[string]string{"role": args[2]})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Roles(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl roles <subcommand> [options]
Subcommands:
  list                     List IAM roles
  create <name>            Create a new role
  assignments              List role assignments
  assign <role> <user>     Assign role to user`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/iam/roles", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "assignments":
		resp, err := apiRequest("GET", "/iam/roles/assignments", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Alerts(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl alerts <subcommand> [options]
Subcommands:
  list             List alert rules
  active           List active alerts
  recent           List recent alerts
  summary          Alert summary
  acknowledge <id> Acknowledge an alert
  resolve <id>     Resolve an alert`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/alerts/rules", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "active":
		resp, err := apiRequest("GET", "/alerts/active", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "recent":
		resp, err := apiRequest("GET", "/alerts/recent", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "summary":
		resp, err := apiRequest("GET", "/alerts/summary", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "acknowledge":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl alerts acknowledge <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/alerts/"+args[1]+"/acknowledge", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "resolve":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl alerts resolve <id>")
			os.Exit(1)
		}
		resp, err := apiRequest("POST", "/alerts/"+args[1]+"/resolve", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Metrics(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl metrics <subcommand> [options]
Subcommands:
  summary    Show metrics summary
  live       Show live metrics stream`)
		return
	}
	resp, err := apiRequest("GET", "/metrics", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	printJSON(result)
}

func Terminal(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl terminal <subcommand> [options]
Subcommands:
  list                List all terminal sessions
  sessions <device>   List sessions for a device
  recording <id>      Get terminal recording`)
		return
	}
	switch args[0] {
	case "list":
		resp, err := apiRequest("GET", "/devices/terminal/sessions", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	case "sessions":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: renctl terminal sessions <device>")
			os.Exit(1)
		}
		resp, err := apiRequest("GET", "/devices/"+args[1]+"/terminal/sessions", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		printJSON(result)
	}
}

func Docker(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl docker <subcommand> [options]
Subcommands:
  containers <device>     List containers on device
  images <device>         List Docker images on device
  volumes <device>        List Docker volumes on device
  networks <device>       List Docker networks on device
  compose <device>        List Compose stacks on device`)
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: renctl docker <subcommand> <device>")
		os.Exit(1)
	}
	deviceID := args[1]
	subpath := ""
	switch args[0] {
	case "containers":
		subpath = "/devices/" + deviceID + "/containers"
	case "images":
		subpath = "/devices/" + deviceID + "/docker-images"
	case "volumes":
		subpath = "/devices/" + deviceID + "/docker-volumes"
	case "networks":
		subpath = "/devices/" + deviceID + "/docker-networks"
	case "compose":
		subpath = "/devices/" + deviceID + "/compose-stacks"
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		os.Exit(1)
	}
	resp, err := apiRequest("GET", subpath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	printJSON(result)
}

func Proxmox(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: renctl proxmox <subcommand> [options]
Subcommands:
  nodes <device>         List Proxmox nodes
  vms <device>           List VMs
  lxcs <device>          List containers
  storage <device>       List storage
  networks <device>      List networks
  snapshots <device>     List snapshots
  tasks <device>         List tasks`)
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: renctl proxmox <subcommand> <device>")
		os.Exit(1)
	}
	deviceID := args[1]
	subpath := "/devices/" + deviceID + "/proxmox/" + args[0]
	resp, err := apiRequest("GET", subpath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	printJSON(result)
}
