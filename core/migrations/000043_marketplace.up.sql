-- Marketplace registry (cached plugin listings)
CREATE TABLE marketplace_plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'general',
    version TEXT NOT NULL DEFAULT '1.0.0',
    min_renos_version TEXT NOT NULL DEFAULT '0.1.0',
    download_url TEXT NOT NULL DEFAULT '',
    homepage TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    screenshots JSONB NOT NULL DEFAULT '[]',
    tags JSONB NOT NULL DEFAULT '[]',
    downloads INTEGER NOT NULL DEFAULT 0,
    rating NUMERIC NOT NULL DEFAULT 0,
    verified BOOLEAN NOT NULL DEFAULT false,
    signature TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL DEFAULT '',
    manifest JSONB NOT NULL DEFAULT '{}',
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_marketplace_plugins_category ON marketplace_plugins(category);
CREATE INDEX idx_marketplace_plugins_name ON marketplace_plugins(name);
CREATE INDEX idx_marketplace_plugins_verified ON marketplace_plugins(verified);

-- Plugin installations (tracks what's installed from marketplace)
CREATE TABLE marketplace_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES marketplace_plugins(id) ON DELETE CASCADE,
    installed_version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'installed' CHECK (status IN ('installed', 'upgrading', 'rolling_back', 'failed')),
    config JSONB NOT NULL DEFAULT '{}',
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_marketplace_installations_plugin ON marketplace_installations(plugin_id);

-- Plugin installation history (version history)
CREATE TABLE marketplace_install_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id UUID NOT NULL REFERENCES marketplace_installations(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('install', 'upgrade', 'rollback')),
    checksum TEXT NOT NULL DEFAULT '',
    verified BOOLEAN NOT NULL DEFAULT false,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_marketplace_install_history_installation ON marketplace_install_history(installation_id);
