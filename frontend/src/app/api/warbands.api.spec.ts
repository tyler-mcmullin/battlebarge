import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { makeWarband } from '../../testing/factories';
import { API_URL } from './api-url';
import { WarbandsApi } from './warbands.api';

describe('WarbandsApi', () => {
  let api: WarbandsApi;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
      ],
    });
    api = TestBed.inject(WarbandsApi);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('lists warbands', () => {
    let result: unknown;
    api.list().subscribe((w) => (result = w));
    const req = http.expectOne('/api/warbands');
    expect(req.request.method).toBe('GET');
    req.flush([makeWarband()]);
    expect(result).toEqual([makeWarband()]);
  });

  it('gets one warband', () => {
    api.get('abc').subscribe();
    const req = http.expectOne('/api/warbands/abc');
    expect(req.request.method).toBe('GET');
    req.flush(makeWarband());
  });

  it('creates with POST /warbands/create', () => {
    api.create({ name: 'Da Boyz', faction: 'Orks' }).subscribe();
    const req = http.expectOne('/api/warbands/create');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ name: 'Da Boyz', faction: 'Orks' });
    req.flush(makeWarband());
  });

  it('updates with PATCH', () => {
    api.update('abc', { name: 'New' }).subscribe();
    const req = http.expectOne('/api/warbands/abc');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ name: 'New' });
    req.flush(makeWarband());
  });

  it('deletes', () => {
    api.delete('abc').subscribe();
    const req = http.expectOne('/api/warbands/abc');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
  });
});
