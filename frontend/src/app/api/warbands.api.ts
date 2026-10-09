import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { API_URL } from './api-url';
import { CreateWarbandRequest, UpdateWarbandRequest, Warband } from './models';

/** Calls for /warbands. Every method returns a cold Observable. */
@Injectable({ providedIn: 'root' })
export class WarbandsApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_URL);

  list() {
    return this.http.get<Warband[]>(`${this.base}/warbands`);
  }

  get(id: string) {
    return this.http.get<Warband>(`${this.base}/warbands/${id}`);
  }

  create(body: CreateWarbandRequest) {
    return this.http.post<Warband>(`${this.base}/warbands/create`, body);
  }

  update(id: string, body: UpdateWarbandRequest) {
    return this.http.patch<Warband>(`${this.base}/warbands/${id}`, body);
  }

  delete(id: string) {
    return this.http.delete<void>(`${this.base}/warbands/${id}`);
  }
}
