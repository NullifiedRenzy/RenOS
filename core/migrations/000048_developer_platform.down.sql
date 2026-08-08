-- M52: Developer Platform (Webhooks, API Keys) (reverse)
-- webhook_deliveries has an FK to webhook_subscriptions, so it is dropped first.

DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_subscriptions;
DROP TABLE IF EXISTS api_keys;
