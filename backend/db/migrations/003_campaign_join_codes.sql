-- Gives every campaign a join code. Joining a campaign now needs the code (the
-- owner does not), so the campaign ID, which is public, is no longer enough.
-- Run once, manually (e.g. psql "$POSTGRES_URL" -f backend/db/migrations/003_campaign_join_codes.sql).

BEGIN;

ALTER TABLE campaigns ADD COLUMN join_code text;

-- Existing campaigns get a random 10-character code (40 bits from Postgres's
-- cryptographically secure gen_random_uuid). Owners can rotate it any time.
UPDATE campaigns
SET join_code = upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 10))
WHERE join_code IS NULL;

ALTER TABLE campaigns ALTER COLUMN join_code SET NOT NULL;

COMMIT;
