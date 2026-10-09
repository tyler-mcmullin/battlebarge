import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { makeUnit } from '../../testing/factories';
import { API_URL } from './api-url';
import { UnitsApi } from './units.api';

describe('UnitsApi', () => {
  let api: UnitsApi;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
      ],
    });
    api = TestBed.inject(UnitsApi);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('creates with POST /units/create', () => {
    api.create({ warband_id: 'wb-1', unit_name: 'Boss' }).subscribe();
    const req = http.expectOne('/api/units/create');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ warband_id: 'wb-1', unit_name: 'Boss' });
    req.flush(makeUnit());
  });

  it('updates and deletes by id', () => {
    api.update('u1', { points: 90 }).subscribe();
    const update = http.expectOne('/api/units/u1');
    expect(update.request.method).toBe('PATCH');
    expect(update.request.body).toEqual({ points: 90 });
    update.flush(makeUnit());

    api.delete('u1').subscribe();
    const del = http.expectOne('/api/units/u1');
    expect(del.request.method).toBe('DELETE');
    del.flush(null, { status: 204, statusText: 'No Content' });
  });

  it('sends kills and xp as {amount}, including negative amounts', () => {
    api.addKills('u1', 1).subscribe();
    const kills = http.expectOne('/api/units/u1/kills');
    expect(kills.request.method).toBe('PATCH');
    expect(kills.request.body).toEqual({ amount: 1 });
    kills.flush(makeUnit());

    api.addXp('u1', -1).subscribe();
    const xp = http.expectOne('/api/units/u1/xp');
    expect(xp.request.body).toEqual({ amount: -1 });
    xp.flush(makeUnit());
  });

  it('adds and removes perks', () => {
    api.addPerk('u1', { name: 'Tough', is_scar: false }).subscribe();
    const add = http.expectOne('/api/units/u1/perk');
    expect(add.request.method).toBe('PATCH');
    expect(add.request.body).toEqual({ name: 'Tough', is_scar: false });
    add.flush(makeUnit());

    api.deletePerk('u1', 'p9').subscribe();
    const remove = http.expectOne('/api/units/u1/perk/p9');
    expect(remove.request.method).toBe('DELETE');
    remove.flush(makeUnit());
  });
});
