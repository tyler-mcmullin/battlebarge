package repositories

// joinCodes
// Generates, checks, and rotates the secret code needed to join a campaign.
// The campaign ID is public, so the code is what actually controls who can join.

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"

	"battlebarge/db"
)

// joinCodeAlphabet leaves out look-alike characters (0/O, 1/I/L) so codes are
// easy to read out loud and type. Ten characters is about 49 bits.
const (
	joinCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	joinCodeLength   = 10
)

// ErrInvalidJoinCode is returned when the join code does not match the campaign's.
var ErrInvalidJoinCode = errors.New("invalid join code")

// Arguments: None
//
// Returns: string - a new random join code; error - if the system's secure random source fails
//
// Generates a join code using crypto/rand, without modulo bias
func NewJoinCode() (string, error) {
	max := big.NewInt(int64(len(joinCodeAlphabet)))
	code := make([]byte, joinCodeLength)
	for i := range code {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = joinCodeAlphabet[n.Int64()]
	}
	return string(code), nil
}

// Arguments: code (string) - a join code as a person might type it
//
// Returns: string - the code uppercased with spaces and hyphens removed
//
// Makes codes match regardless of case or grouping, e.g. "k7m2-x9qd-4t" and "K7M2X9QD4T"
func NormalizeJoinCode(code string) string {
	code = strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(code))
	return strings.ToUpper(code)
}

// Arguments: campaignID (string) - campaign ID
//
// Returns: string - the campaign's join code; error - pgx.ErrNoRows if the campaign does not exist
//
// Fetches a campaign's join code. Only call this for the campaign's owner
func GetJoinCode(campaignID string) (string, error) {
	var code string
	err := db.PGClient.QueryRow(context.Background(),
		`SELECT join_code FROM campaigns WHERE id = $1`, campaignID).Scan(&code)
	return code, err
}

// Arguments: campaignID (string) - campaign ID; ownerID (string) - ID of the owning user
//
// Returns: string - the new join code; error - pgx.ErrNoRows if no campaign matched the ID and owner
//
// Replaces a campaign's join code with a new random one, so the old code stops working
func RotateJoinCode(campaignID string, ownerID string) (string, error) {
	code, err := NewJoinCode()
	if err != nil {
		return "", err
	}

	tag, err := db.PGClient.Exec(context.Background(),
		`UPDATE campaigns SET join_code = $1, updated_at = now() WHERE id = $2 AND owner_id = $3`,
		code, campaignID, ownerID)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", pgx.ErrNoRows
	}

	return code, nil
}

// Arguments: given (string) - the code the caller supplied; actual (string) - the campaign's stored code
//
// Returns: bool - true if they match after normalizing
//
// Compares join codes in constant time so response timing does not reveal how much of a guess was right
func joinCodeMatches(given string, actual string) bool {
	a := []byte(NormalizeJoinCode(given))
	b := []byte(NormalizeJoinCode(actual))
	return subtle.ConstantTimeCompare(a, b) == 1
}
