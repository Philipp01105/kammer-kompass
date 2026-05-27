-- +goose Up
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

DROP TRIGGER IF EXISTS trg_ihks_updated_at ON ihks;
CREATE TRIGGER trg_ihks_updated_at
BEFORE UPDATE ON ihks
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_ihks_slug ON ihks(slug);
CREATE INDEX IF NOT EXISTS idx_ihks_state ON ihks(state);

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

DROP TRIGGER IF EXISTS trg_ihk_info_pages_updated_at ON ihk_info_pages;
CREATE TRIGGER trg_ihk_info_pages_updated_at
BEFORE UPDATE ON ihk_info_pages
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

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

-- +goose Down
DROP TABLE IF EXISTS ihk_info_versions;
DROP TABLE IF EXISTS ihk_info_pages;
DROP TABLE IF EXISTS ihks;

