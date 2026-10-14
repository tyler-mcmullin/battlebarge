-- Gives every user their own UUID instead of using the Firebase UID as the
-- primary key. The Firebase UID moves to users.firebase_id (unique), and
-- warbands.user_id and campaigns.owner_id are repointed at the new UUIDs.
-- Run once, manually (e.g. psql "$POSTGRES_URL" -f backend/db/migrations/005_user_uuid_ids.sql).
-- Take a pg_dump backup first. Needs Postgres 13+ (gen_random_uuid).
-- Deploy the matching backend right after: the old code expects text IDs.

BEGIN;

-- keep the Firebase UID, then mint a UUID for every existing user
ALTER TABLE users ADD COLUMN firebase_id varchar(128);
UPDATE users SET firebase_id = id;
ALTER TABLE users ALTER COLUMN firebase_id SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_firebase_id_key UNIQUE (firebase_id);
ALTER TABLE users ADD COLUMN new_id uuid NOT NULL DEFAULT gen_random_uuid();

-- warbands.user_id
ALTER TABLE warbands DROP CONSTRAINT IF EXISTS warbands_user_id_fkey;
ALTER TABLE warbands ADD COLUMN new_user_id uuid;
UPDATE warbands w SET new_user_id = u.new_id FROM users u WHERE u.id = w.user_id;
ALTER TABLE warbands ALTER COLUMN new_user_id SET NOT NULL;
ALTER TABLE warbands DROP COLUMN user_id;
ALTER TABLE warbands RENAME COLUMN new_user_id TO user_id;

-- campaigns.owner_id
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_owner_id_fkey;
ALTER TABLE campaigns ADD COLUMN new_owner_id uuid;
UPDATE campaigns c SET new_owner_id = u.new_id FROM users u WHERE u.id = c.owner_id;
ALTER TABLE campaigns ALTER COLUMN new_owner_id SET NOT NULL;
ALTER TABLE campaigns DROP COLUMN owner_id;
ALTER TABLE campaigns RENAME COLUMN new_owner_id TO owner_id;
CREATE INDEX campaigns_owner_id_idx ON campaigns (owner_id);

-- swap the users primary key
ALTER TABLE users DROP CONSTRAINT users_pkey;
ALTER TABLE users DROP COLUMN id;
ALTER TABLE users RENAME COLUMN new_id TO id;
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE users ADD PRIMARY KEY (id);

ALTER TABLE warbands ADD CONSTRAINT warbands_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);
ALTER TABLE campaigns ADD CONSTRAINT campaigns_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users(id);

COMMIT;
