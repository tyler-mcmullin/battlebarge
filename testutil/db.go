// Package testutil holds helpers shared by the project's tests.
package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"battlebarge/db"
	"battlebarge/models"
)

// schemaSQL mirrors the tables the repositories expect. The users, warbands,
// units and unit_perks tables are copied from a pg_dump of the real database;
// the campaign tables come from db/migrations/002_campaigns.sql plus
// 003_campaign_join_codes.sql. The repo keeps
// no schema file, so keep this in sync when the real schema changes.
const schemaSQL = `
CREATE TABLE users (
	id         varchar(128) PRIMARY KEY,
	email      varchar(255) NOT NULL UNIQUE,
	username   varchar(50) NOT NULL UNIQUE,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE warbands (
	id                 uuid PRIMARY KEY,
	user_id            text NOT NULL REFERENCES users(id),
	name               text NOT NULL,
	faction            text NOT NULL DEFAULT '',
	description        text NOT NULL DEFAULT '',
	requisition_points integer NOT NULL DEFAULT 0,
	supply_limit       integer NOT NULL DEFAULT 0,
	created_at         timestamptz NOT NULL DEFAULT now(),
	updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE units (
	id             uuid PRIMARY KEY,
	warband_id     uuid NOT NULL REFERENCES warbands(id) ON DELETE CASCADE,
	unit_name      text NOT NULL,
	narrative_name text NOT NULL DEFAULT '',
	bio            text NOT NULL DEFAULT '',
	points         integer NOT NULL DEFAULT 0,
	kills          integer NOT NULL DEFAULT 0,
	experience     integer NOT NULL DEFAULT 0,
	created_at     timestamptz NOT NULL DEFAULT now(),
	updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE unit_perks (
	id          uuid PRIMARY KEY,
	unit_id     uuid NOT NULL REFERENCES units(id) ON DELETE CASCADE,
	name        text NOT NULL,
	description text NOT NULL DEFAULT '',
	is_scar     boolean NOT NULL DEFAULT false,
	created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX unit_perks_unit_id_idx ON unit_perks (unit_id);

CREATE TABLE campaigns (
	id                   uuid PRIMARY KEY,
	owner_id             varchar(128) NOT NULL REFERENCES users(id),
	name                 text NOT NULL,
	description          text NOT NULL DEFAULT '',
	points_per_win       integer NOT NULL DEFAULT 0,
	points_per_loss      integer NOT NULL DEFAULT 0,
	starting_requisition integer NOT NULL DEFAULT 0,
	join_code            text NOT NULL,
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
`

// Arguments: t (*testing.T) - the running test
//
// Returns: None
//
// Points db.PGClient at a throwaway schema in the Postgres database named by
// TEST_POSTGRES_URL, creating the tables the repositories need. The schema is
// dropped when the test finishes. The test is skipped if TEST_POSTGRES_URL is
// not set, so the suite still passes without a database.
func SetupDB(t *testing.T) {
	t.Helper()

	url := os.Getenv("TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("TEST_POSTGRES_URL not set; skipping database test")
	}

	ctx := context.Background()
	schema := "test_" + uuid.NewString()[:8]

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create schema: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		admin.Close()
		t.Fatalf("parse TEST_POSTGRES_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 32 // enough that concurrency tests can really overlap

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		admin.Close()
		t.Fatalf("connect pool: %v", err)
	}
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		pool.Close()
		admin.Close()
		t.Fatalf("create tables: %v", err)
	}

	prev := db.PGClient
	db.PGClient = pool

	t.Cleanup(func() {
		pool.Close()
		admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
		db.PGClient = prev
	})
}

// Arguments: t (*testing.T) - the running test; id (string) - Firebase-style user ID
//
// Returns: models.User - the inserted user
//
// Inserts a user directly into the test database; the email and username are derived from the ID
func InsertUser(t *testing.T, id string) models.User {
	t.Helper()

	now := time.Now()
	u := models.User{
		ID:        id,
		Email:     id + "@example.com",
		Username:  "user_" + id,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.PGClient.Exec(context.Background(),
		`INSERT INTO users (id, email, username, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Email, u.Username, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return u
}

// Arguments: t (*testing.T) - the running test; userID (string) - owning user ID; name (string) - warband name
//
// Returns: models.Warband - the inserted warband (without computed fields)
//
// Inserts a warband directly into the test database
func InsertWarband(t *testing.T, userID, name string) models.Warband {
	t.Helper()

	now := time.Now()
	w := models.Warband{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.PGClient.Exec(context.Background(),
		`INSERT INTO warbands (id, user_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		w.ID, w.UserID, w.Name, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		t.Fatalf("insert warband: %v", err)
	}
	return w
}

// Arguments: t (*testing.T) - the running test; warbandID (uuid.UUID) - owning warband; name (string) - unit name; points (int) - point cost
//
// Returns: models.Unit - the inserted unit (without perks)
//
// Inserts a unit directly into the test database
func InsertUnit(t *testing.T, warbandID uuid.UUID, name string, points int) models.Unit {
	t.Helper()

	now := time.Now()
	u := models.Unit{
		ID:        uuid.New(),
		WarbandID: warbandID,
		UnitName:  name,
		Points:    points,
		Perks:     []models.Perk{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.PGClient.Exec(context.Background(),
		`INSERT INTO units (id, warband_id, unit_name, points, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		u.ID, u.WarbandID, u.UnitName, u.Points, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		t.Fatalf("insert unit: %v", err)
	}
	return u
}

// Arguments: t (*testing.T) - the running test; ownerID (string) - owning user ID; name (string) - campaign name
//
// Returns: models.Campaign - the inserted campaign (without chapters, teams, or warbands), with JoinCode set to "TESTCODE01"
//
// Inserts a campaign directly into the test database
func InsertCampaign(t *testing.T, ownerID, name string) models.Campaign {
	t.Helper()

	now := time.Now()
	c := models.Campaign{ID: uuid.New(), OwnerID: ownerID, Name: name, JoinCode: "TESTCODE01", CreatedAt: now, UpdatedAt: now}
	_, err := db.PGClient.Exec(context.Background(),
		`INSERT INTO campaigns (id, owner_id, name, join_code, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.OwnerID, c.Name, c.JoinCode, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		t.Fatalf("insert campaign: %v", err)
	}
	return c
}

// Arguments: t (*testing.T) - the running test; campaignID (uuid.UUID) - owning campaign; name (string) - team name
//
// Returns: models.CampaignTeam - the inserted team
//
// Inserts a campaign team directly into the test database
func InsertTeam(t *testing.T, campaignID uuid.UUID, name string) models.CampaignTeam {
	t.Helper()

	now := time.Now()
	team := models.CampaignTeam{ID: uuid.New(), CampaignID: campaignID, Name: name, CreatedAt: now, UpdatedAt: now}
	_, err := db.PGClient.Exec(context.Background(),
		`INSERT INTO campaign_teams (id, campaign_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		team.ID, team.CampaignID, team.Name, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		t.Fatalf("insert team: %v", err)
	}
	return team
}
