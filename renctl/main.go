package main

import (
	"fmt"
	"os"

	"github.com/renos-project/renos/renctl/cmd"
)

var version = "0.1.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "login":
		cmd.Login(args[1:])
	case "logout":
		cmd.Logout()
	case "devices":
		cmd.Devices(args[1:])
	case "jobs":
		cmd.Jobs(args[1:])
	case "workflows":
		cmd.Workflows(args[1:])
	case "plugins":
		cmd.Plugins(args[1:])
	case "cluster":
		cmd.Cluster(args[1:])
	case "vault":
		cmd.Vault(args[1:])
	case "users":
		cmd.Users(args[1:])
	case "roles":
		cmd.Roles(args[1:])
	case "alerts":
		cmd.Alerts(args[1:])
	case "metrics":
		cmd.Metrics(args[1:])
	case "terminal":
		cmd.Terminal(args[1:])
	case "docker":
		cmd.Docker(args[1:])
	case "proxmox":
		cmd.Proxmox(args[1:])
	case "version":
		fmt.Println("renctl version", version)
	case "completion":
		printCompletion(args[1:])
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`renctl - RenOS Command Line Interface

Usage:
  renctl <command> [arguments]

Commands:
  login         Authenticate with RenOS server
  logout        Clear stored credentials
  devices       Manage devices (list, get, tag, exec)
  jobs          Manage jobs (list, get, cancel, retry)
  workflows     Manage workflows (list, create, run)
  plugins       Manage plugins (list, install, remove)
  cluster       Manage cluster (nodes, tasks, state)
  vault         Manage secrets (list, get, create, delete)
  users         Manage users (list, update role)
  roles         Manage IAM roles (list, create, assign)
  alerts        Manage alerts (list, acknowledge, resolve)
  metrics       View metrics (summary, live)
  terminal      Open terminal sessions
  docker        Manage Docker resources
  proxmox       Manage Proxmox resources
  version       Print version
  completion    Generate shell completion

Run 'renctl <command> --help' for more information.`)
}

func printCompletion(args []string) {
	shell := "bash"
	if len(args) > 0 {
		shell = args[0]
	}
	switch shell {
	case "bash":
		fmt.Println(`_renctl_completions() {
  local cur prev words cword
  _init_completion || return
  commands="login logout devices jobs workflows plugins cluster vault users roles alerts metrics terminal docker proxmox version completion"
  COMPREPLY=($(compgen -W "$commands" -- "$cur"))
}
complete -F _renctl_completions renctl`)
	case "zsh":
		fmt.Println(`#compdef renctl
_renctl() {
  local -a commands
  commands=(
    'login:Authenticate with RenOS server'
    'logout:Clear stored credentials'
    'devices:Manage devices'
    'jobs:Manage jobs'
    'workflows:Manage workflows'
    'plugins:Manage plugins'
    'cluster:Manage cluster'
    'vault:Manage secrets'
    'users:Manage users'
    'roles:Manage IAM roles'
    'alerts:Manage alerts'
    'metrics:View metrics'
    'terminal:Open terminal sessions'
    'docker:Manage Docker resources'
    'proxmox:Manage Proxmox resources'
    'version:Print version'
    'completion:Generate shell completion'
  )
  _describe 'renctl' commands
}
compdef _renctl renctl`)
	case "fish":
		fmt.Println(`complete -c renctl -f
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a login -d "Authenticate with RenOS server"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a logout -d "Clear stored credentials"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a devices -d "Manage devices"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a jobs -d "Manage jobs"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a workflows -d "Manage workflows"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a plugins -d "Manage plugins"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a cluster -d "Manage cluster"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a vault -d "Manage secrets"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a users -d "Manage users"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a roles -d "Manage IAM roles"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a alerts -d "Manage alerts"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a metrics -d "View metrics"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a terminal -d "Open terminal sessions"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a docker -d "Manage Docker resources"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a proxmox -d "Manage Proxmox resources"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a version -d "Print version"
complete -c renctl -n "not __fish_seen_subcommand_from $commands" -a completion -d "Generate shell completion"`)
	default:
		fmt.Fprintf(os.Stderr, "unsupported shell: %s (supported: bash, zsh, fish)\n", shell)
		os.Exit(1)
	}
}
