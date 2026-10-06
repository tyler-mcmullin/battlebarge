-- Makes usernames unique ignoring case, so "Alice" and "alice" cannot both
-- exist. The original capitalization is kept for display.
-- Run once, manually (e.g. psql "$POSTGRES_URL" -f db/migrations/004_username_case_insensitive.sql).

BEGIN;

-- Fail with a readable message, instead of an index error, if existing
-- usernames already collide when lowercased. Resolve those by hand first.
DO $$
DECLARE
	dupes text;
BEGIN
	SELECT string_agg(name || ' (' || n || ' users)', ', ')
	INTO dupes
	FROM (
		SELECT lower(username) AS name, count(*) AS n
		FROM users
		GROUP BY lower(username)
		HAVING count(*) > 1
	) t;

	IF dupes IS NOT NULL THEN
		RAISE EXCEPTION 'usernames collide ignoring case, rename these first: %', dupes;
	END IF;
END $$;

CREATE UNIQUE INDEX users_username_lower_key ON users (lower(username));

COMMIT;
