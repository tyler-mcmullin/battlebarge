-- Adds campaigns, their chapters, teams, and warband memberships.
-- Run once, manually (e.g. psql "$POSTGRES_URL" -f backend/db/migrations/002_campaigns.sql).

BEGIN;

CREATE TABLE campaigns (
	id                   uuid PRIMARY KEY,
	owner_id             varchar(128) NOT NULL REFERENCES users(id),
	name                 text NOT NULL,
	description          text NOT NULL DEFAULT '',
	points_per_win       integer NOT NULL DEFAULT 0,
	points_per_loss      integer NOT NULL DEFAULT 0,
	starting_requisition integer NOT NULL DEFAULT 0,
	created_at           timestamptz NOT NULL DEFAULT now(),
	updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX campaigns_owner_id_idx ON campaigns (owner_id);

CREATE TABLE campaign_chapters (
	id          uuid PRIMARY KEY,
	campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	title       text NOT NULL,
	description text NOT NULL DEFAULT '',
	sort_order  integer NOT NULL DEFAULT 0,
	created_at  timestamptz NOT NULL DEFAULT now(),
	updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX campaign_chapters_campaign_id_idx ON campaign_chapters (campaign_id);

CREATE TABLE campaign_teams (
	id          uuid PRIMARY KEY,
	campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	name        text NOT NULL,
	created_at  timestamptz NOT NULL DEFAULT now(),
	updated_at  timestamptz NOT NULL DEFAULT now(),
	UNIQUE (campaign_id, name),
	-- lets campaign_warbands prove a team belongs to the same campaign
	UNIQUE (id, campaign_id)
);

-- A warband can be in many campaigns, once each, on exactly one team per
-- campaign. A team with warbands on it cannot be deleted (no ON DELETE
-- action on the team FK).
CREATE TABLE campaign_warbands (
	campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	warband_id  uuid NOT NULL REFERENCES warbands(id) ON DELETE CASCADE,
	team_id     uuid NOT NULL,
	joined_at   timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (campaign_id, warband_id),
	FOREIGN KEY (team_id, campaign_id) REFERENCES campaign_teams (id, campaign_id)
);

CREATE INDEX campaign_warbands_warband_id_idx ON campaign_warbands (warband_id);

COMMIT;
