-- M52: Developer Platform (Webhooks, API Keys)

-- Webhook subscriptions
CREATE TABLE webhook_subscriptions (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url          TEXT NOT NULL,
    events       JSONB NOT NULL DEFAULT '[]',
    secret       TEXT NOT NULL,
    active       BOOLEAN NOT NULL DEFAULT true,
    headers      JSONB NOT NULL DEFAULT '{}',
    retry_policy INTEGER NOT NULL DEFAULT 3,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_webhook_sub_user ON webhook_subscriptions(user_id);
CREATE INDEX idx_webhook_sub_events ON webhook_subscriptions USING gin(events);
CREATE INDEX idx_webhook_sub_active ON webhook_subscriptions(active);

-- Webhook deliveries (audit log)
CREATE TABLE webhook_deliveries (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    subscription_id TEXT NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    event           TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    status_code     INTEGER NOT NULL DEFAULT 0,
    response        TEXT NOT NULL DEFAULT '',
    success         BOOLEAN NOT NULL DEFAULT false,
    retry_count     INTEGER NOT NULL DEFAULT 0,
    delivered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_webhook_del_sub ON webhook_deliveries(subscription_id);
CREATE INDEX idx_webhook_del_event ON webhook_deliveries(event, delivered_at DESC);

-- Developer API keys (separate from IAM tokens)
CREATE TABLE api_keys (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    key_prefix   TEXT NOT NULL,
    scopes       JSONB NOT NULL DEFAULT '[]',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    active       BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_key_user ON api_keys(user_id);
CREATE INDEX idx_api_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_key_active ON api_keys(active);
