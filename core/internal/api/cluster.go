package api

import (
	"encoding/json"
	"net/http"

	"github.com/renos-project/renos/core/internal/db"
)

func clusterDashboardHandler(nodes *db.ClusterNodeRepository, metrics *db.ClusterMetricsRepository, tasks *db.ClusterTaskRepository, elections *db.ClusterElectionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allNodes, _ := nodes.ListAll(r.Context())
		leader, _ := nodes.FindLeader(r.Context())
		leaderID := ""
		if leader != nil { leaderID = leader.NodeID }
		term, _ := elections.CurrentTerm(r.Context())
		pendingTasks, _ := tasks.ListPending(r.Context(), 1000)
		nodeMetrics, _ := metrics.LatestByNode(r.Context())

		summary := map[string]interface{}{
			"nodes":         allNodes,
			"leader_id":     leaderID,
			"term":          term,
			"pending_tasks": len(pendingTasks),
			"metrics":       nodeMetrics,
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

func listClusterNodesHandler(nodes *db.ClusterNodeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := nodes.ListAll(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list nodes"); return }
		if list == nil { list = []*db.ClusterNode{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func registerClusterNodeHandler(nodes *db.ClusterNodeRepository) http.HandlerFunc {
	type request struct {
		NodeID   string `json:"node_id"`
		Hostname string `json:"hostname"`
		Address  string `json:"address"`
		Port     int    `json:"port"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		if req.NodeID == "" { req.NodeID = "auto" }
		if req.Port == 0 { req.Port = 8443 }

		node := &db.ClusterNode{
			NodeID:   req.NodeID,
			Hostname: req.Hostname,
			Address:  req.Address,
			Port:     req.Port,
			Role:     "follower",
			Status:   "active",
		}
		if err := nodes.Register(r.Context(), node); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to register: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, node)
	}
}

func updateClusterNodeStatusHandler(nodes *db.ClusterNodeRepository) http.HandlerFunc {
	type request struct {
		Status string `json:"status"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.URL.Query().Get("node_id")
		var req request
		json.NewDecoder(r.Body).Decode(&req)
		if nodeID == "" || req.Status == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "node_id and status required")
			return
		}
		if err := nodes.UpdateStatus(r.Context(), nodeID, req.Status); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func listClusterTasksHandler(tasks *db.ClusterTaskRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		var list []*db.ClusterTask
		var err error
		if status == "pending" {
			list, err = tasks.ListPending(r.Context(), 50)
		} else {
			list, err = tasks.ListAll(r.Context(), 50)
		}
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list tasks"); return }
		if list == nil { list = []*db.ClusterTask{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createClusterTaskHandler(tasks *db.ClusterTaskRepository) http.HandlerFunc {
	type request struct {
		TaskType   string          `json:"task_type"`
		TaskData   json.RawMessage `json:"task_data"`
		Priority   int             `json:"priority"`
		MaxRetries int             `json:"max_retries"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		if req.TaskType == "" { writeError(w, http.StatusBadRequest, "invalid_request", "task_type required"); return }
		if req.TaskData == nil { req.TaskData = json.RawMessage(`{}`) }
		if req.MaxRetries == 0 { req.MaxRetries = 3 }

		task := &db.ClusterTask{TaskType: req.TaskType, TaskData: req.TaskData, Priority: req.Priority, MaxRetries: req.MaxRetries, Status: "pending"}
		if err := tasks.Create(r.Context(), task); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create task: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	}
}

func rebalanceClusterTasksHandler(tasks *db.ClusterTaskRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := tasks.Rebalance(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		writeJSON(w, http.StatusOK, map[string]int64{"rebalanced": count})
	}
}

func clusterStateHandler(state *db.ClusterStateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, _ := state.ListAll(r.Context())
		if list == nil { list = []*db.ClusterState{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func setClusterStateHandler(state *db.ClusterStateRepository) http.HandlerFunc {
	type request struct {
		Key        string          `json:"key"`
		Value      json.RawMessage `json:"value"`
		Owner      string          `json:"owner"`
		TTLSeconds int             `json:"ttl_seconds"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		if req.Key == "" { writeError(w, http.StatusBadRequest, "invalid_request", "key required"); return }
		if req.Value == nil { req.Value = json.RawMessage(`{}`) }

		s := &db.ClusterState{Key: req.Key, Value: req.Value, Owner: req.Owner, TTLSeconds: req.TTLSeconds}
		if err := state.Set(r.Context(), s); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s)
	}
}

func clusterElectionsHandler(elections *db.ClusterElectionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, _ := elections.ListRecent(r.Context(), 20)
		if list == nil { list = []*db.ClusterElection{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func recordClusterMetricsHandler(metrics *db.ClusterMetricsRepository) http.HandlerFunc {
	type request struct {
		NodeID            string  `json:"node_id"`
		CPUUsage          float64 `json:"cpu_usage"`
		MemoryUsage       float64 `json:"memory_usage"`
		DiskUsage         float64 `json:"disk_usage"`
		ActiveConnections int     `json:"active_connections"`
		TaskCount         int     `json:"task_count"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		m := &db.ClusterMetric{
			NodeID:            req.NodeID,
			CPUUsage:          req.CPUUsage,
			MemoryUsage:       req.MemoryUsage,
			DiskUsage:         req.DiskUsage,
			ActiveConnections: req.ActiveConnections,
			TaskCount:         req.TaskCount,
		}
		if err := metrics.Record(r.Context(), m); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, m)
	}
}
