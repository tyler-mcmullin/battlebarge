import type { components, paths } from './schema';

/**
 * Friendly names for the API's types. They come from `schema.d.ts`, which is
 * generated from docs/openapi.yaml (`npm run api:generate`), so they always
 * match what the Go API really sends.
 */
type Schemas = components['schemas'];

export type User = Schemas['User'];
export type Perk = Schemas['Perk'];
export type Unit = Schemas['Unit'];
export type Warband = Schemas['Warband'];

export type RegisterRequest = Schemas['RegisterRequest'];
export type CreateWarbandRequest = Schemas['CreateWarbandRequest'];
export type UpdateWarbandRequest = Schemas['UpdateWarbandRequest'];
export type CreateUnitRequest = Schemas['CreateUnitRequest'];
export type UpdateUnitRequest = Schemas['UpdateUnitRequest'];
export type AddPerkRequest = Schemas['AddPerkRequest'];

export type RegisterResponse =
  paths['/auth/register']['post']['responses']['201']['content']['application/json'];
