-- In-app notifications for the notification center (M66)
CREATE TABLE IF NOT EXISTS inapp_notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL DEFAULT 'system',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id   TEXT NOT NULL DEFAULT '',
    metadata    JSONB NOT NULL DEFAULT '{}',
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inapp_notifications_user_read
    ON inapp_notifications (user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_inapp_notifications_user_created
    ON inapp_notifications (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inapp_notifications_user_type
    ON inapp_notifications (user_id, type);
