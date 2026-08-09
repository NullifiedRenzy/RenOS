-- installed_plugins: central plugin registry (not per-device)
CREATE TABLE installed_plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '0.1.0',
    author TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    plugin_type TEXT NOT NULL DEFAULT 'go' CHECK (plugin_type IN ('go', 'wasm', 'container', 'script')),
    status TEXT NOT NULL DEFAULT 'installed' CHECK (status IN ('installed', 'enabled', 'disabled', 'error', 'uninstalled')),
    error_message TEXT NOT NULL DEFAULT '',
    permissions JSONB NOT NULL DEFAULT '[]',
    config JSONB NOT NULL DEFAULT '{}',
    config_schema JSONB NOT NULL DEFAULT '{}',
    dependencies JSONB NOT NULL DEFAULT '[]',
    entry_point TEXT NOT NULL DEFAULT '',
    blob_path TEXT NOT NULL DEFAULT '',
    blob_hash TEXT NOT NULL DEFAULT '',
    signature TEXT NOT NULL DEFAULT '',
    enabled_at TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,
    last_error_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_installed_plugins_name ON installed_plugins(name);
CREATE INDEX idx_installed_plugins_status ON installed_plugins(status);

-- plugin_routes: API routes provided by plugins
CREATE TABLE plugin_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES installed_plugins(id) ON DELETE CASCADE,
    method TEXT NOT NULL DEFAULT 'GET' CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')),
    path TEXT NOT NULL,
    handler_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    require_auth BOOLEAN NOT NULL DEFAULT true,
    min_role TEXT NOT NULL DEFAULT 'viewer',
    rate_limit INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, method, path)
);

CREATE INDEX idx_plugin_routes_plugin ON plugin_routes(plugin_id);

-- plugin_pages: Hub pages provided by plugins
CREATE TABLE plugin_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES installed_plugins(id) ON DELETE CASCADE,
    page_path TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT 'puzzle',
    component_name TEXT NOT NULL DEFAULT '',
    parent_path TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    require_auth BOOLEAN NOT NULL DEFAULT true,
    min_role TEXT NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, page_path)
);

CREATE INDEX idx_plugin_pages_plugin ON plugin_pages(plugin_id);

-- plugin_sidebar: sidebar entries provided by plugins
CREATE TABLE plugin_sidebar (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES installed_plugins(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    icon TEXT NOT NULL DEFAULT 'puzzle',
    path TEXT NOT NULL,
    parent_label TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    min_role TEXT NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, path)
);

CREATE INDEX idx_plugin_sidebar_plugin ON plugin_sidebar(plugin_id);

-- plugin_workers: background workers provided by plugins
CREATE TABLE plugin_workers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES installed_plugins(id) ON DELETE CASCADE,
    worker_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    handler_name TEXT NOT NULL DEFAULT '',
    concurrency INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('running', 'stopped', 'error')),
    last_error TEXT NOT NULL DEFAULT '',
    restart_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, worker_name)
);

CREATE INDEX idx_plugin_workers_plugin ON plugin_workers(plugin_id);

-- plugin_schedules: scheduled tasks provided by plugins
CREATE TABLE plugin_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES installed_plugins(id) ON DELETE CASCADE,
    schedule_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    handler_name TEXT NOT NULL DEFAULT '',
    cron_expr TEXT NOT NULL DEFAULT '',
    interval_seconds INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    last_status TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plugin_id, schedule_name)
);

CREATE INDEX idx_plugin_schedules_plugin ON plugin_schedules(plugin_id);
