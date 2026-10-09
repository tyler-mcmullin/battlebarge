import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { ApiError } from '../api/api-error';
import { API_URL } from '../api/api-url';
import { FakeAuthClient } from '../../testing/fake-auth-client';
import { apiInterceptor } from './api.interceptor';
import { AuthClient } from './auth-client';
import { AuthService } from './auth.service';
import { Notify } from './notify';

describe('apiInterceptor', () => {
  let client: FakeAuthClient;
  let http: HttpTestingController;
  let httpClient: HttpClient;
  let router: Router;
  let navigate: ReturnType<typeof vi.spyOn>;
  const notify = { info: vi.fn(), error: vi.fn() };

  beforeEach(async () => {
    notify.info.mockReset();
    notify.error.mockReset();
    client = new FakeAuthClient();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([apiInterceptor])),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: AuthClient, useValue: client },
        { provide: Notify, useValue: notify },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    httpClient = TestBed.inject(HttpClient);
    router = TestBed.inject(Router);
    navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    await TestBed.inject(AuthService).login('a@example.com', 'secret123'); // signed in by default
    client.calls.length = 0;
  });

  afterEach(() => http.verify());

  it('sends the bearer token', async () => {
    httpClient.get('/api/warbands').subscribe();
    let header: string | null = null;
    await vi.waitFor(() => {
      const req = http.expectOne('/api/warbands');
      header = req.request.headers.get('Authorization');
      req.flush([]);
    });
    expect(header).toBe('Bearer test-token');
  });

  it('sends no Authorization header when signed out', async () => {
    await TestBed.inject(AuthService).logout();
    httpClient.get('/api/warbands/abc').subscribe();
    let header: string | null = 'unset';
    await vi.waitFor(() => {
      const req = http.expectOne('/api/warbands/abc');
      header = req.request.headers.get('Authorization');
      req.flush({});
    });
    expect(header).toBeNull();
  });

  it('leaves other URLs alone: no token is sent anywhere else', () => {
    httpClient.get('https://example.com/data').subscribe();
    const req = http.expectOne('https://example.com/data');
    expect(req.request.headers.has('Authorization')).toBe(false);
    req.flush({});
  });

  it('on 401 signs out, explains, and goes to the login page', async () => {
    let received: unknown;
    httpClient.get('/api/warbands').subscribe({ error: (e) => (received = e) });
    await vi.waitFor(() =>
      http
        .expectOne('/api/warbands')
        .flush({ error: 'token revoked' }, { status: 401, statusText: 'Unauthorized' }),
    );

    await vi.waitFor(() => expect(navigate).toHaveBeenCalled());
    expect(client.calls).toContain('signOut');
    expect(notify.error).toHaveBeenCalledWith('Your session was ended. Please sign in again.');
    expect(navigate).toHaveBeenCalledWith(
      ['/login'],
      expect.objectContaining({ queryParams: expect.anything() }),
    );
    expect(received).toBeInstanceOf(ApiError);
    expect((received as ApiError).handled).toBe(true);
  });

  it('on 401 for a disabled account, says so', async () => {
    httpClient.get('/api/warbands').subscribe({ error: () => undefined });
    await vi.waitFor(() =>
      http
        .expectOne('/api/warbands')
        .flush({ error: 'account disabled' }, { status: 401, statusText: 'Unauthorized' }),
    );
    await vi.waitFor(() =>
      expect(notify.error).toHaveBeenCalledWith('This account has been disabled.'),
    );
  });

  it('on 403 "email not verified" goes to the verification page', async () => {
    let received: unknown;
    httpClient.get('/api/warbands').subscribe({ error: (e) => (received = e) });
    await vi.waitFor(() =>
      http
        .expectOne('/api/warbands')
        .flush({ error: 'email not verified' }, { status: 403, statusText: 'Forbidden' }),
    );
    await vi.waitFor(() => expect(navigate).toHaveBeenCalledWith(['/verify-email']));
    expect((received as ApiError).handled).toBe(true);
  });

  it('leaves other 403s for the screen to report (for example a wrong join code)', async () => {
    let received: unknown;
    httpClient.get('/api/warbands').subscribe({ error: (e) => (received = e) });
    await vi.waitFor(() =>
      http
        .expectOne('/api/warbands')
        .flush({ error: 'invalid join code' }, { status: 403, statusText: 'Forbidden' }),
    );
    await vi.waitFor(() => expect(received).toBeInstanceOf(ApiError));
    expect((received as ApiError).handled).toBe(false);
    expect((received as ApiError).message).toBe('invalid join code');
    expect(navigate).not.toHaveBeenCalled();
  });

  it('on 429 tells the user how long to wait', async () => {
    httpClient.get('/api/warbands').subscribe({ error: () => undefined });
    await vi.waitFor(() =>
      http
        .expectOne('/api/warbands')
        .flush(
          { error: 'too many requests' },
          { status: 429, statusText: 'Too Many Requests', headers: { 'Retry-After': '12' } },
        ),
    );
    await vi.waitFor(() =>
      expect(notify.error).toHaveBeenCalledWith('Too many requests. Try again in 12 seconds.'),
    );
  });

  it('passes ordinary failures through untouched for the screen to show', async () => {
    let received: unknown;
    httpClient.get('/api/warbands').subscribe({ error: (e) => (received = e) });
    await vi.waitFor(() =>
      http
        .expectOne('/api/warbands')
        .flush(
          { error: 'limit reached: at most 50 warbands per user' },
          { status: 409, statusText: 'Conflict' },
        ),
    );
    await vi.waitFor(() => expect(received).toBeInstanceOf(ApiError));
    expect((received as ApiError).status).toBe(409);
    expect((received as ApiError).handled).toBe(false);
    expect(notify.error).not.toHaveBeenCalled();
  });
});
