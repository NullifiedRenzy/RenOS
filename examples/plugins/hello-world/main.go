package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	greeting      = "Hello from RenOS!"
	greetCount    atomic.Uint64
	workerRuns    atomic.Uint64
	lastWorkerRun atomic.Value
)

func init() {
	lastWorkerRun.Store(time.Now().Format(time.RFC3339))
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/plugins/hello-world/greet", handleGreet)
	mux.HandleFunc("POST /api/v1/plugins/hello-world/greet", handleGreetPost)
	mux.HandleFunc("GET /api/v1/plugins/hello-world/stats", handleStats)

	go workerLoop()

	addr := ":8090"
	fmt.Printf("[hello-world] plugin listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("[hello-world] server error: %v\n", err)
	}
}

func handleGreet(w http.ResponseWriter, r *http.Request) {
	greetCount.Add(1)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": greeting,
	})
}

func handleGreetPost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
		return
	}
	greetCount.Add(1)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Hello, %s! Welcome to RenOS.", req.Name),
	})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	lastRun, _ := lastWorkerRun.Load().(string)
	writeJSON(w, http.StatusOK, map[string]any{
		"greet_count":     greetCount.Load(),
		"worker_runs":     workerRuns.Load(),
		"last_worker_run": lastRun,
		"uptime_seconds":  int(time.Now().Unix()),
	})
}

func workerLoop() {
	for {
		workerRuns.Add(1)
		now := time.Now().Format(time.RFC3339)
		lastWorkerRun.Store(now)
		fmt.Printf("[hello-world] worker tick at %s (total runs: %d)\n", now, workerRuns.Load())
		time.Sleep(60 * time.Second)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
