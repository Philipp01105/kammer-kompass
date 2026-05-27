-- +goose Up
DROP TABLE IF EXISTS ihk_proposals;

ALTER TABLE ihks
DROP COLUMN IF EXISTS created_from_proposal_id;

ALTER TABLE ihk_info_versions
DROP COLUMN IF EXISTS based_on_ihk_proposal_id;

ALTER TABLE review_events
DROP CONSTRAINT IF EXISTS review_events_target_type_check;

DELETE FROM review_events
WHERE target_type = 'ihk_proposal';

ALTER TABLE review_events
ADD CONSTRAINT review_events_target_type_check
CHECK (target_type IN ('info_suggestion'));

-- +goose Down
ALTER TABLE review_events
DROP CONSTRAINT IF EXISTS review_events_target_type_check;

ALTER TABLE review_events
ADD CONSTRAINT review_events_target_type_check
CHECK (target_type IN ('info_suggestion', 'ihk_proposal'));
