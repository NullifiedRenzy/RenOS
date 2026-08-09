package db

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SearchResult is a single lightweight hit across any searchable entity.
type SearchResult struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Href     string `json:"href"`
}

// SearchResponse groups results by entity type.
type SearchResponse struct {
	Query   string                   `json:"query"`
	Total   int                      `json:"total"`
	Results map[string][]SearchResult `json:"results"`
}

// SearchRepository runs fast ILIKE queries across all searchable tables.
type SearchRepository struct {
	pool *pgxpool.Pool
}

func NewSearchRepository(pool *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{pool: pool}
}

// Search matches q against names/hostnames/titles/descriptions/ids across
// devices, jobs, workflows, plugins, policies, secrets, users, logs,
// containers, VMs and cluster nodes. Values are never searched for secrets.
// Per-category limit keeps the response snappy.
func (r *SearchRepository) Search(ctx context.Context, q string, limit int) (*SearchResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	pattern := "%" + q + "%"
	resp := &SearchResponse{
		Query:   q,
		Results: map[string][]SearchResult{},
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	appendResult := func(category string, res []SearchResult) {
		mu.Lock()
		defer mu.Unlock()
		if len(res) > 0 {
			resp.Results[category] = res
			resp.Total += len(res)
		}
	}

	run := func(category, query string, args ...any) {
		defer wg.Done()
		rows, err := r.pool.Query(ctx, query, args...)
		if err != nil {
			return
		}
		defer rows.Close()
		var out []SearchResult
		for rows.Next() {
			var res SearchResult
			if err := rows.Scan(&res.ID, &res.Title, &res.Subtitle, &res.Href); err != nil {
				return
			}
			res.Type = category
			out = append(out, res)
		}
		appendResult(category, out)
	}

	// devices
	wg.Add(1)
	run("devices",
		`SELECT id, hostname, COALESCE(os, '') || ' · ' || COALESCE(status, ''), '/devices/' || id
		 FROM devices
		 WHERE hostname ILIKE $1 OR COALESCE(labels,'') ILIKE $1 OR COALESCE(metadata,'') ILIKE $1
		 ORDER BY hostname LIMIT $2`, pattern, limit)

	// jobs
	wg.Add(1)
	run("jobs",
		`SELECT id, COALESCE(type, 'job') || ' #' || substr(id::text, 1, 8), COALESCE(status, ''), '/jobs/' || id
		 FROM jobs
		 WHERE id::text ILIKE $1 OR COALESCE(type,'') ILIKE $1 OR COALESCE(status,'') ILIKE $1
		 ORDER BY created_at DESC LIMIT $2`, pattern, limit)

	// workflows
	wg.Add(1)
	run("workflows",
		`SELECT id, name, COALESCE(description, ''), '/workflows'
		 FROM workflows
		 WHERE name ILIKE $1 OR COALESCE(description,'') ILIKE $1
		 ORDER BY name LIMIT $2`, pattern, limit)

	// installed plugins
	wg.Add(1)
	run("plugins",
		`SELECT id, COALESCE(display_name, name), COALESCE(author, '') || ' · v' || COALESCE(version, '0'), '/plugins'
		 FROM installed_plugins
		 WHERE name ILIKE $1 OR COALESCE(display_name,'') ILIKE $1 OR COALESCE(author,'') ILIKE $1 OR COALESCE(description,'') ILIKE $1
		 ORDER BY name LIMIT $2`, pattern, limit)

	// marketplace plugins
	wg.Add(1)
	run("marketplace",
		`SELECT id, COALESCE(display_name, name), COALESCE(author, '') || ' · v' || COALESCE(version, '0'), '/marketplace'
		 FROM marketplace_plugins
		 WHERE name ILIKE $1 OR COALESCE(display_name,'') ILIKE $1 OR COALESCE(author,'') ILIKE $1 OR COALESCE(description,'') ILIKE $1
		 ORDER BY name LIMIT $2`, pattern, limit)

	// policies
	wg.Add(1)
	run("policies",
		`SELECT id, name, COALESCE(description, '') || ' · ' || COALESCE(action, ''), '/policies/' || id
		 FROM policies
		 WHERE name ILIKE $1 OR COALESCE(description,'') ILIKE $1
		 ORDER BY name LIMIT $2`, pattern, limit)

	// vault secrets (names/paths only — never values)
	wg.Add(1)
	run("secrets",
		`SELECT id, name, COALESCE(path, ''), '/vault'
		 FROM vault_secrets
		 WHERE name ILIKE $1 OR COALESCE(path,'') ILIKE $1 OR COALESCE(description,'') ILIKE $1
		 ORDER BY name LIMIT $2`, pattern, limit)

	// users
	wg.Add(1)
	run("users",
		`SELECT id, COALESCE(display_name, email), COALESCE(email, '') || ' · ' || COALESCE(role, ''), '/users'
		 FROM users
		 WHERE email ILIKE $1 OR COALESCE(display_name,'') ILIKE $1
		 ORDER BY email LIMIT $2`, pattern, limit)

	// logs (system + monitoring)
	wg.Add(1)
	run("logs",
		`SELECT id, COALESCE(message, ''), COALESCE(level, '') || ' · ' || COALESCE(source, ''), '/logs'
		 FROM monitoring_logs
		 WHERE message ILIKE $1 OR COALESCE(source,'') ILIKE $1 OR COALESCE(level,'') ILIKE $1
		 ORDER BY recorded_at DESC LIMIT $2`, pattern, limit)

	// containers
	wg.Add(1)
	run("containers",
		`SELECT id, name, COALESCE(image, ''), '/devices/' || COALESCE(device_id, '') || '/docker'
		 FROM containers
		 WHERE name ILIKE $1 OR COALESCE(image,'') ILIKE $1
		 ORDER BY name LIMIT $2`, pattern, limit)

	// virtual machines (libvirt + proxmox)
	wg.Add(1)
	run("vms",
		`SELECT id, domain_name, COALESCE(os_type, '') || ' · ' || COALESCE(state, ''), '/devices/' || COALESCE(device_id, '') || '/proxmox'
		 FROM virtual_machines
		 WHERE domain_name ILIKE $1
		 ORDER BY domain_name LIMIT $2`, pattern, limit)

	wg.Add(1)
	run("proxmox-vms",
		`SELECT id::text, name, COALESCE(node_name, '') || ' · ' || COALESCE(status, ''), '/cluster'
		 FROM proxmox_vms
		 WHERE name ILIKE $1
		 ORDER BY name LIMIT $2`, pattern, limit)

	// cluster nodes
	wg.Add(1)
	run("clusters",
		`SELECT id, COALESCE(hostname, node_id), COALESCE(address, '') || ' · ' || COALESCE(role, ''), '/cluster'
		 FROM cluster_nodes
		 WHERE hostname ILIKE $1 OR COALESCE(node_id,'') ILIKE $1 OR COALESCE(address,'') ILIKE $1
		 ORDER BY hostname LIMIT $2`, pattern, limit)

	wg.Wait()
	return resp, nil
}
