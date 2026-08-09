package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	renossdk "github.com/renos-project/renos/sdk/go"
)

func main() {
	email := os.Getenv("RENOS_EMAIL")
	password := os.Getenv("RENOS_PASSWORD")
	server := os.Getenv("RENOS_URL")

	if email == "" || password == "" {
		log.Fatal("RENOS_EMAIL and RENOS_PASSWORD must be set")
	}

	client, err := renossdk.NewClientFromLogin(server, email, password)
	if err != nil {
		log.Fatalf("login failed: %v", err)
	}
	fmt.Println("connected to RenOS")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = ctx

	devices, err := client.ListDevices(1, 10)
	if err != nil {
		log.Fatalf("list devices failed: %v", err)
	}
	fmt.Printf("found %d devices\n", devices.Total)
	for _, d := range devices.Items {
		fmt.Printf("  - %s (%s) [%s]\n", d.Name, d.Hostname, d.Status)
	}

	if len(devices.Items) > 0 {
		device := devices.Items[0]
		fmt.Printf("\nrunning command on %s...\n", device.Hostname)
		cmd, err := client.ExecuteCommand(device.ID, "uname -a")
		if err != nil {
			log.Fatalf("execute command failed: %v", err)
		}
		fmt.Printf("command %s: %s\n", cmd.ID, cmd.Status)

		time.Sleep(2 * time.Second)

		cmdResult, err := client.GetCommand(device.ID, cmd.ID)
		if err != nil {
			log.Fatalf("get command failed: %v", err)
		}
		fmt.Printf("output: %s\n", cmdResult.Output)
	}

	workflows, err := client.ListWorkflows(1, 20)
	if err != nil {
		log.Fatalf("list workflows failed: %v", err)
	}
	fmt.Printf("\nfound %d workflows:\n", workflows.Total)
	for _, w := range workflows.Items {
		fmt.Printf("  - %s (enabled: %v)\n", w.Name, w.Enabled)
	}

	jobs, err := client.ListJobs(1, 10)
	if err != nil {
		log.Fatalf("list jobs failed: %v", err)
	}
	fmt.Printf("\nrecent jobs: %d total\n", jobs.Total)
	for _, j := range jobs.Items {
		fmt.Printf("  - %s [%s] %d/%d\n", j.Type, j.Status, j.Completed, j.Total)
	}

	alerts, err := client.ListAlerts(1, 10)
	if err != nil {
		log.Printf("list alerts failed: %v", err)
	} else {
		fmt.Printf("\nactive alerts: %d\n", alerts.Total)
		for _, a := range alerts.Items {
			fmt.Printf("  - %s (%s) [%s]\n", a.Name, a.Severity, a.Status)
		}
	}

	users, err := client.ListUsers(1, 20)
	if err != nil {
		log.Printf("list users failed: %v", err)
	} else {
		fmt.Printf("\nusers: %d\n", users.Total)
		for _, u := range users.Items {
			fmt.Printf("  - %s (%s) [%s]\n", u.Email, u.DisplayName, u.Role)
		}
	}

	groups, err := client.ListGroups(1, 20)
	if err != nil {
		log.Printf("list groups failed: %v", err)
	} else {
		fmt.Printf("\ngroups: %d\n", groups.Total)
		for _, g := range groups.Items {
			fmt.Printf("  - %s (%d devices)\n", g.Name, g.DeviceCount)
		}
	}

	schedules, err := client.ListSchedules(1, 20)
	if err != nil {
		log.Printf("list schedules failed: %v", err)
	} else {
		fmt.Printf("\nschedules: %d\n", schedules.Total)
		for _, s := range schedules.Items {
			fmt.Printf("  - %s (%s)\n", s.Name, s.CronExpression)
		}
	}

	fmt.Println("\ndone")
}
