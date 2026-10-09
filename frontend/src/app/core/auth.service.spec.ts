import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { API_URL } from '../api/api-url';
import { FakeAuthClient } from '../../testing/fake-auth-client';
import { AuthClient } from './auth-client';
import { AuthService } from './auth.service';

describe('AuthService', () => {
  let client: FakeAuthClient;
  let http: HttpTestingController;

  function setup(): AuthService {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: AuthClient, useValue: client },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    return TestBed.inject(AuthService);
  }

  beforeEach(() => {
    client = new FakeAuthClient();
  });

  it('starts signed out and is ready once Firebase has reported', async () => {
    const auth = setup();
    await auth.whenReady();
    expect(auth.isSignedIn()).toBe(false);
    expect(auth.emailVerified()).toBe(false);
  });

  it('restores a saved session', async () => {
    client.user = { uid: 'u1', email: 'a@example.com', emailVerified: true };
    const auth = setup();
    await auth.whenReady();
    expect(auth.isSignedIn()).toBe(true);
    expect(auth.emailVerified()).toBe(true);
    expect(auth.user()?.email).toBe('a@example.com');
  });

  it('logs in and out', async () => {
    const auth = setup();
    await auth.login('a@example.com', 'secret123');
    expect(auth.isSignedIn()).toBe(true);
    expect(client.calls).toEqual(['signIn:a@example.com']);

    await auth.logout();
    expect(auth.isSignedIn()).toBe(false);
    expect(client.calls).toContain('signOut');
  });

  it('leaves the user signed out when login fails', async () => {
    client.signInError = Object.assign(new Error('bad'), { code: 'auth/invalid-credential' });
    const auth = setup();
    await expect(auth.login('a@example.com', 'nope')).rejects.toMatchObject({
      code: 'auth/invalid-credential',
    });
    expect(auth.isSignedIn()).toBe(false);
  });

  it('registers with the API, then signs in, then sends the verification email', async () => {
    client.signInResult = { uid: 'u1', email: 'a@example.com', emailVerified: false };
    const auth = setup();
    const request = { email: 'a@example.com', username: 'alice', password: 'secret123' };

    const done = auth.register(request);
    const req = http.expectOne('/api/auth/register');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(request);
    expect(client.calls).toEqual([]); // nothing happens in Firebase until the API accepts the account
    req.flush(
      { message: 'user created', user_id: 'u1', email_verification_required: true },
      { status: 201, statusText: 'Created' },
    );
    await done;

    expect(client.calls).toEqual(['signIn:a@example.com', 'sendVerificationEmail']);
    expect(auth.isSignedIn()).toBe(true);
    expect(auth.emailVerified()).toBe(false);
  });

  it('does not sign in when the API refuses the registration', async () => {
    const auth = setup();
    const done = auth.register({
      email: 'a@example.com',
      username: 'alice',
      password: 'secret123',
    });
    http
      .expectOne('/api/auth/register')
      .flush({ error: 'username already taken' }, { status: 409, statusText: 'Conflict' });
    await expect(done).rejects.toBeTruthy();
    expect(client.calls).toEqual([]);
    expect(auth.isSignedIn()).toBe(false);
  });

  it('picks up verification when the account is refreshed', async () => {
    client.user = { uid: 'u1', email: 'a@example.com', emailVerified: false };
    const auth = setup();
    await auth.whenReady();
    expect(auth.emailVerified()).toBe(false);

    client.refreshResult = { uid: 'u1', email: 'a@example.com', emailVerified: true };
    expect(await auth.refreshVerification()).toBe(true);
    expect(auth.emailVerified()).toBe(true);
  });

  it('reports not verified yet when the account is still unverified', async () => {
    client.user = { uid: 'u1', email: 'a@example.com', emailVerified: false };
    const auth = setup();
    await auth.whenReady();
    expect(await auth.refreshVerification()).toBe(false);
  });

  it('hands out the ID token only while signed in', async () => {
    const auth = setup();
    expect(await auth.idToken()).toBeNull();
    await auth.login('a@example.com', 'secret123');
    expect(await auth.idToken()).toBe('test-token');
  });
});
