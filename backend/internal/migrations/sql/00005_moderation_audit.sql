-- +goose Up
CREATE TABLE IF NOT EXISTS moderation_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term TEXT NOT NULL,
    normalized_term TEXT NOT NULL,
    category TEXT NOT NULL CHECK (
        category IN ('insult', 'slur', 'threat', 'sexual', 'spam', 'other')
    ),
    severity TEXT NOT NULL CHECK (
        severity IN ('low', 'medium', 'high')
    ),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (normalized_term, category)
);

DROP TRIGGER IF EXISTS trg_moderation_terms_updated_at ON moderation_terms;
CREATE TRIGGER trg_moderation_terms_updated_at
BEFORE UPDATE ON moderation_terms
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_moderation_terms_normalized
ON moderation_terms(normalized_term)
WHERE is_active = true;

CREATE TABLE IF NOT EXISTS review_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type TEXT NOT NULL CHECK (
        target_type IN ('info_suggestion')
    ),
    target_id UUID NOT NULL,
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    old_status TEXT,
    new_status TEXT,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_review_events_target ON review_events(target_type, target_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    scope_type TEXT,
    scope_id TEXT,
    old_value JSONB,
    new_value JSONB,
    ip_hash TEXT,
    user_agent_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS review_events;
DROP TABLE IF EXISTS moderation_terms;

