-- Schema for sqlc. Keep in sync with goose migrations.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS role_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    allow_mask BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    role_template_id UUID NOT NULL REFERENCES role_templates(id),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('global', 'state', 'ihk')),
    scope_id TEXT,
    allow_mask BIGINT NOT NULL,
    deny_mask BIGINT NOT NULL DEFAULT 0,
    granted_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ihks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    city TEXT,
    state TEXT NOT NULL,
    official_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ihk_info_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ihk_id UUID NOT NULL UNIQUE REFERENCES ihks(id),
    current_text TEXT NOT NULL DEFAULT '',
    confidence_level TEXT NOT NULL DEFAULT 'low'
        CHECK (confidence_level IN ('low', 'medium', 'high')),
    source_summary TEXT,
    last_version_id UUID,
    locked BOOLEAN NOT NULL DEFAULT false,
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ihk_info_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ihk_info_page_id UUID NOT NULL REFERENCES ihk_info_pages(id),
    ihk_id UUID NOT NULL REFERENCES ihks(id),
    version_number INT NOT NULL,
    old_text TEXT NOT NULL,
    new_text TEXT NOT NULL,
    change_summary TEXT NOT NULL,
    changed_by UUID REFERENCES users(id),
    based_on_info_suggestion_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ihk_info_page_id, version_number)
);

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

CREATE TABLE IF NOT EXISTS permission_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_type TEXT NOT NULL CHECK (request_type IN ('registration', 'role_request')),
    requested_role_template_id UUID NOT NULL REFERENCES role_templates(id),
    requested_scope_type TEXT NOT NULL CHECK (requested_scope_type IN ('global', 'state', 'ihk')),
    requested_scope_id TEXT,
    proof_note TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    decision_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_permission_requests_pending_scope
ON permission_requests (
    user_id,
    requested_role_template_id,
    requested_scope_type,
    COALESCE(requested_scope_id, '')
)
WHERE status = 'pending';
