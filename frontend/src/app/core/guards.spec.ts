import { TestBed } from '@angular/core/testing';
import {
  ActivatedRouteSnapshot,
  provideRouter,
  Router,
  RouterStateSnapshot,
  UrlTree,
} from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { FakeAuthClient } from '../../testing/fake-auth-client';
import { API_URL } from '../api/api-url';
import { AuthClient, AuthUser } from './auth-client';
import { authGuard, guestGuard, verifiedGuard } from './guards';

describe('route guards', () => {
  let router: Router;

  function setup(user: AuthUser | null) {
    const client = new FakeAuthClient();
    client.user = user;
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: AuthClient, useValue: client },
      ],
    });
    router = TestBed.inject(Router);
  }

  const route = {} as ActivatedRouteSnapshot;
  const state = (url: string) => ({ url }) as RouterStateSnapshot;
  const run = (guard: typeof authGuard, url = '/warbands/abc') =>
    TestBed.runInInjectionContext(() => guard(route, state(url)));

  const signedOut = null;
  const unverified: AuthUser = { uid: 'u1', email: 'a@example.com', emailVerified: false };
  const verified: AuthUser = { uid: 'u1', email: 'a@example.com', emailVerified: true };

  describe('authGuard', () => {
    it('lets a signed-in user through, verified or not', async () => {
      setup(unverified);
      expect(await run(authGuard)).toBe(true);
    });

    it('sends a signed-out user to login and remembers where they were going', async () => {
      setup(signedOut);
      const result = (await run(authGuard, '/warbands/abc')) as UrlTree;
      expect(router.serializeUrl(result)).toBe('/login?returnUrl=%2Fwarbands%2Fabc');
    });
  });

  describe('verifiedGuard', () => {
    it('lets a verified user through', async () => {
      setup(verified);
      expect(await run(verifiedGuard)).toBe(true);
    });

    it('sends an unverified user to the verification page', async () => {
      setup(unverified);
      expect(router.serializeUrl((await run(verifiedGuard)) as UrlTree)).toBe('/verify-email');
    });

    it('sends a signed-out user to login', async () => {
      setup(signedOut);
      expect(router.serializeUrl((await run(verifiedGuard)) as UrlTree)).toContain('/login');
    });
  });

  describe('guestGuard', () => {
    it('lets a signed-out user see the login page', async () => {
      setup(signedOut);
      expect(await run(guestGuard)).toBe(true);
    });

    it('moves signed-in users along: verified to the app, unverified to verification', async () => {
      setup(verified);
      expect(router.serializeUrl((await run(guestGuard)) as UrlTree)).toBe('/warbands');
      TestBed.resetTestingModule();
      setup(unverified);
      expect(router.serializeUrl((await run(guestGuard)) as UrlTree)).toBe('/verify-email');
    });
  });
});
