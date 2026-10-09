import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { API_URL } from './api-url';
import { AddPerkRequest, CreateUnitRequest, Unit, UpdateUnitRequest } from './models';

/** Calls for /units. Every method returns a cold Observable. */
@Injectable({ providedIn: 'root' })
export class UnitsApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_URL);

  get(id: string) {
    return this.http.get<Unit>(`${this.base}/units/${id}`);
  }

  create(body: CreateUnitRequest) {
    return this.http.post<Unit>(`${this.base}/units/create`, body);
  }

  update(id: string, body: UpdateUnitRequest) {
    return this.http.patch<Unit>(`${this.base}/units/${id}`, body);
  }

  delete(id: string) {
    return this.http.delete<void>(`${this.base}/units/${id}`);
  }

  /** Adds to the unit's kills; a negative amount subtracts (it never goes below 0). */
  addKills(id: string, amount: number) {
    return this.http.patch<Unit>(`${this.base}/units/${id}/kills`, { amount });
  }

  /** Adds to the unit's experience; a negative amount subtracts (it never goes below 0). */
  addXp(id: string, amount: number) {
    return this.http.patch<Unit>(`${this.base}/units/${id}/xp`, { amount });
  }

  addPerk(id: string, body: AddPerkRequest) {
    return this.http.patch<Unit>(`${this.base}/units/${id}/perk`, body);
  }

  deletePerk(unitId: string, perkId: string) {
    return this.http.delete<Unit>(`${this.base}/units/${unitId}/perk/${perkId}`);
  }
}
