-- Moves units.perks (JSON column) into its own unit_perks table.
-- Run once, manually (e.g. psql "$POSTGRES_URL" -f db/migrations/001_unit_perks_table.sql).
-- Assumes units.perks is json/jsonb holding an array of
-- {"id": uuid, "name": text, "description": text, "is_scar": bool}.
-- If it is a text column, change `perks` to `perks::jsonb` below.

BEGIN;

CREATE TABLE unit_perks (
	id          uuid PRIMARY KEY,
	unit_id     uuid NOT NULL REFERENCES units(id) ON DELETE CASCADE,
	name        text NOT NULL,
	description text NOT NULL DEFAULT '',
	is_scar     boolean NOT NULL DEFAULT false,
	created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX unit_perks_unit_id_idx ON unit_perks (unit_id);

-- Preserve existing perk order via created_at offsets (1 microsecond apart).
INSERT INTO unit_perks (id, unit_id, name, description, is_scar, created_at)
SELECT (p.elem ->> 'id')::uuid,
       u.id,
       p.elem ->> 'name',
       COALESCE(p.elem ->> 'description', ''),
       COALESCE((p.elem ->> 'is_scar')::boolean, false),
       now() + (p.ord * interval '1 microsecond')
FROM units u,
     LATERAL jsonb_array_elements(u.perks) WITH ORDINALITY AS p(elem, ord)
WHERE u.perks IS NOT NULL;

-- Optional safety check: before COMMIT, compare
--   SELECT sum(jsonb_array_length(perks)) FROM units;  SELECT count(*) FROM unit_perks;
-- and ROLLBACK if they differ. (Run the file in an open psql session to do this.)
ALTER TABLE units DROP COLUMN perks;

COMMIT;
