import { Perk, Unit, Warband } from '../app/api/models';

export function makePerk(overrides: Partial<Perk> = {}): Perk {
  return { id: 'perk-1', name: 'Tough', description: 'Hard to kill', is_scar: false, ...overrides };
}

export function makeUnit(overrides: Partial<Unit> = {}): Unit {
  return {
    id: 'unit-1',
    warband_id: 'wb-1',
    unit_name: 'Boss',
    narrative_name: 'Grukk',
    bio: '',
    points: 80,
    kills: 2,
    experience: 5,
    perks: [],
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

export function makeWarband(overrides: Partial<Warband> = {}): Warband {
  return {
    id: 'wb-1',
    user_id: 'u1',
    name: 'Da Boyz',
    faction: 'Orks',
    description: '',
    requisition_points: 5,
    supply_limit: 1000,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    units: [],
    num_units: 0,
    total_points_cost: 0,
    crusade_points: 0,
    ...overrides,
  };
}
