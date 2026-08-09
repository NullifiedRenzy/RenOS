-- M51: Mobile Platform APIs

-- Push notification device registrations (FCM, APNs)
CREATE TABLE push_registrations (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform     TEXT NOT NULL CHECK (platform IN ('fcm', 'apns', 'webpush')),
    token        TEXT NOT NULL,
    device_name  TEXT NOT NULL DEFAULT '',
    device_model TEXT NOT NULL DEFAULT '',
    os_version   TEXT NOT NULL DEFAULT '',
    app_version  TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, platform, token)
);
CREATE INDEX idx_push_reg_user ON push_registrations(user_id);
CREATE INDEX idx_push_reg_platform ON push_registrations(platform, enabled);

-- Mobile sessions (for biometric re-auth, session management)
CREATE TABLE mobile_sessions (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id       TEXT NOT NULL,
    device_name     TEXT NOT NULL DEFAULT '',
    device_model    TEXT NOT NULL DEFAULT '',
    os_version      TEXT NOT NULL DEFAULT '',
    app_version     TEXT NOT NULL DEFAULT '',
    ip_address      TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    biometric_enabled BOOLEAN NOT NULL DEFAULT false,
    push_enabled    BOOLEAN NOT NULL DEFAULT false,
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mobile_session_user ON mobile_sessions(user_id);
CREATE INDEX idx_mobile_session_device ON mobile_sessions(device_id);
CREATE INDEX idx_mobile_session_expires ON mobile_sessions(expires_at);

-- Offline sync queue (commands/state queued while device is offline)
CREATE TABLE offline_sync_queue (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id    TEXT NOT NULL,
    action       TEXT NOT NULL,
    payload      JSONB NOT NULL DEFAULT '{}',
    priority     INTEGER NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'expired')),
    retry_count  INTEGER NOT NULL DEFAULT 0,
    max_retries  INTEGER NOT NULL DEFAULT 3,
    error_message TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);
CREATE INDEX idx_offline_sync_user_device ON offline_sync_queue(user_id, device_id);
CREATE INDEX idx_offline_sync_status ON offline_sync_queue(status, priority DESC, created_at);

-- Mobile device approval requests
CREATE TABLE device_approvals (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id    TEXT NOT NULL,
    device_name  TEXT NOT NULL DEFAULT '',
    device_model TEXT NOT NULL DEFAULT '',
    ip_address   TEXT NOT NULL DEFAULT '',
    fingerprint  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    approved_by  UUID REFERENCES users(id),
    approved_at  TIMESTAMPTZ,
    reject_reason TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_device_approval_status ON device_approvals(status);
CREATE INDEX idx_device_approval_user ON device_approvals(user_id);

-- Mobile dashboard summaries (cached/precomputed)
CREATE TABLE mobile_summaries (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    summary_type TEXT NOT NULL CHECK (summary_type IN ('overview', 'devices', 'alerts', 'deploy', 'cluster')),
    data        JSONB NOT NULL DEFAULT '{}',
    version     INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, summary_type)
);
CREATE INDEX idx_mobile_summary_user ON mobile_summaries(user_id);
