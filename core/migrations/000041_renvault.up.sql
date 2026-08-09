-- Vault secrets (encrypted at rest)
CREATE TABLE vault_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL UNIQUE,
    secret_type TEXT NOT NULL DEFAULT 'generic' CHECK (secret_type IN ('generic', 'password', 'api_key', 'certificate', 'ssh_key', 'token')),
    encrypted_value BYTEA NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vault_secrets_path ON vault_secrets(path);
CREATE INDEX idx_vault_secrets_type ON vault_secrets(secret_type);

-- Vault secret versions (history)
CREATE TABLE vault_secret_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    secret_id UUID NOT NULL REFERENCES vault_secrets(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    encrypted_value BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    changed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vault_secret_versions_secret ON vault_secret_versions(secret_id);
CREATE UNIQUE INDEX idx_vault_secret_versions_unique ON vault_secret_versions(secret_id, version);

-- Vault encryption keys
CREATE TABLE vault_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    algorithm TEXT NOT NULL DEFAULT 'aes-256-gcm',
    public_key TEXT NOT NULL DEFAULT '',
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Vault access policies
CREATE TABLE vault_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    paths JSONB NOT NULL DEFAULT '[]',
    capabilities JSONB NOT NULL DEFAULT '["read"]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Vault access grants
CREATE TABLE vault_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES vault_policies(id) ON DELETE CASCADE,
    user_id UUID,
    role TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vault_grants_policy ON vault_grants(policy_id);
CREATE INDEX idx_vault_grants_user ON vault_grants(user_id);

-- Vault audit log
CREATE TABLE vault_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    secret_id UUID,
    action TEXT NOT NULL,
    user_id UUID,
    ip_address TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vault_audit_log_secret ON vault_audit_log(secret_id);
CREATE INDEX idx_vault_audit_log_created ON vault_audit_log(created_at DESC);
