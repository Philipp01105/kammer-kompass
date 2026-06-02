-- +goose Up

-- Replace all-bits-set sentinel (-1) with explicit union of known permission bits.
-- SyncRoleTemplates will keep these in sync with AllPermissions in rbac/masks.go.
UPDATE role_templates
SET allow_mask = 8589934591
WHERE allow_mask < 0;

UPDATE user_role_assignments
SET allow_mask = 8589934591
WHERE allow_mask < 0;

ALTER TABLE role_templates
    ADD CONSTRAINT chk_role_templates_allow_mask_known CHECK (allow_mask >= 0 AND allow_mask <= 8589934591);

ALTER TABLE user_role_assignments
    ADD CONSTRAINT chk_ura_allow_mask_known CHECK (allow_mask >= 0 AND allow_mask <= 8589934591),
    ADD CONSTRAINT chk_ura_deny_mask_known  CHECK (deny_mask  >= 0 AND deny_mask  <= 8589934591);

-- +goose Down
ALTER TABLE role_templates
    DROP CONSTRAINT IF EXISTS chk_role_templates_allow_mask_known;

ALTER TABLE user_role_assignments
    DROP CONSTRAINT IF EXISTS chk_ura_allow_mask_known,
    DROP CONSTRAINT IF EXISTS chk_ura_deny_mask_known;
