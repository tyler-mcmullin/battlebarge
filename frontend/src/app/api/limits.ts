/**
 * Validation limits, mirrored from the API (see the schemas in docs/openapi.yaml
 * and the binding tags in backend/models/models.go). The API enforces them; the
 * forms use the same numbers so people get feedback before sending a request.
 */
export const LIMITS = {
  username: 50,
  email: 255,
  passwordMin: 8,
  passwordMax: 128,
  name: 100,
  faction: 100,
  warbandDescription: 2000,
  unitBio: 5000,
  perkDescription: 1000,
  points: { min: 0, max: 1_000_000 },
} as const;
