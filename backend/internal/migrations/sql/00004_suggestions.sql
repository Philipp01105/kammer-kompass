-- +goose Up
CREATE TABLE IF NOT EXISTS info_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ihk_id UUID NOT NULL REFERENCES ihks(id),
    current_text_snapshot TEXT NOT NULL,

    suggested_change TEXT NOT NULL,
    public_pending_text TEXT NOT NULL,

    reason TEXT,
    source_url TEXT,
    source_note TEXT,

    language_code TEXT NOT NULL DEFAULT 'de',
    language_confidence NUMERIC(4,3) NOT NULL DEFAULT 0,
    pre_moderation_status TEXT NOT NULL CHECK (
        pre_moderation_status IN (
            'passed',
            'blocked_language',
            'blocked_word_filter',
            'blocked_html',
            'blocked_url',
            'blocked_length'
        )
    ),

    moderation_flags JSONB NOT NULL DEFAULT '[]'::jsonb,

    public_pending_visible BOOLEAN NOT NULL DEFAULT false,
    public_pending_created_at TIMESTAMPTZ,
    public_pending_hidden_at TIMESTAMPTZ,
    public_pending_hidden_by UUID REFERENCES users(id),
    public_pending_hide_reason TEXT,

    submitted_by_user_id UUID REFERENCES users(id),
    submitted_email TEXT,
    ip_hash TEXT,

    status TEXT NOT NULL CHECK (
        status IN (
            'submitted',
            'under_review',
            'needs_more_info',
            'accepted',
            'rejected',
            'applied',
            'archived',
            'spam'
        )
    ),

    assigned_to UUID REFERENCES users(id),
    accepted_by UUID REFERENCES users(id),
    applied_by UUID REFERENCES users(id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS trg_info_suggestions_updated_at ON info_suggestions;
CREATE TRIGGER trg_info_suggestions_updated_at
BEFORE UPDATE ON info_suggestions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_info_suggestions_status ON info_suggestions(status);
CREATE INDEX IF NOT EXISTS idx_info_suggestions_ihk_id ON info_suggestions(ihk_id);
CREATE INDEX IF NOT EXISTS idx_info_suggestions_public_pending
ON info_suggestions(ihk_id, public_pending_visible, status, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS info_suggestions;

