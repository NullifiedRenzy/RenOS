-- Monitoring metrics
CREATE TABLE monitoring_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    metric_value NUMERIC NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitoring_metrics_device ON monitoring_metrics(device_id);
CREATE INDEX idx_monitoring_metrics_name ON monitoring_metrics(metric_name);
CREATE INDEX idx_monitoring_metrics_recorded ON monitoring_metrics(recorded_at);
CREATE INDEX idx_monitoring_metrics_device_name_time ON monitoring_metrics(device_id, metric_name, recorded_at DESC);

-- Alert rules
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metric_name TEXT NOT NULL,
    condition TEXT NOT NULL DEFAULT 'above' CHECK (condition IN ('above', 'below', 'equals', 'not_equals')),
    threshold NUMERIC NOT NULL DEFAULT 0,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('info', 'warning', 'critical')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    notification_channels JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Alert instances
CREATE TABLE alert_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'firing' CHECK (status IN ('firing', 'resolved', 'silenced')),
    value NUMERIC NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    acknowledged_by UUID,
    acknowledged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_instances_rule ON alert_instances(rule_id);
CREATE INDEX idx_alert_instances_status ON alert_instances(status);
CREATE INDEX idx_alert_instances_device ON alert_instances(device_id);
CREATE INDEX idx_alert_instances_firing ON alert_instances(status) WHERE status = 'firing';

-- Monitoring dashboards
CREATE TABLE monitoring_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    layout JSONB NOT NULL DEFAULT '[]',
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dashboard widgets
CREATE TABLE monitoring_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL REFERENCES monitoring_dashboards(id) ON DELETE CASCADE,
    widget_type TEXT NOT NULL DEFAULT 'chart' CHECK (widget_type IN ('chart', 'stat', 'table', 'gauge', 'log')),
    title TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    position_x INTEGER NOT NULL DEFAULT 0,
    position_y INTEGER NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 6,
    height INTEGER NOT NULL DEFAULT 4,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitoring_widgets_dashboard ON monitoring_widgets(dashboard_id);

-- Monitoring logs
CREATE TABLE monitoring_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    source TEXT NOT NULL DEFAULT 'system',
    level TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warn', 'error', 'fatal')),
    message TEXT NOT NULL,
    fields JSONB NOT NULL DEFAULT '{}',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitoring_logs_device ON monitoring_logs(device_id);
CREATE INDEX idx_monitoring_logs_level ON monitoring_logs(level);
CREATE INDEX idx_monitoring_logs_recorded ON monitoring_logs(recorded_at DESC);
CREATE INDEX idx_monitoring_logs_source ON monitoring_logs(source);
