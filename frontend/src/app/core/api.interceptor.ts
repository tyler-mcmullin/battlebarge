import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, from, switchMap, throwError } from 'rxjs';
import { ApiError } from '../api/api-error';
import { API_URL } from '../api/api-url';
import { AuthService } from './auth.service';
import { Notify } from './notify';

const SESSION_MESSAGES: Record<string, string> = {
  'token revoked': 'Your session was ended. Please sign in again.',
  'account disabled': 'This account has been disabled.',
};

/**
 * For calls to our API: adds the Firebase ID token, and handles the failures
 * every screen would otherwise have to handle itself.
 *  - 401: the session is no good, so sign out and go to the login page.
 *  - 403 "email not verified": go to the verification page.
 *  - 429: too many requests; tell the user how long to wait.
 * The error is still passed on (as an ApiError, marked `handled` when we already
 * told the user) so screens can stop their spinners.
 */
export const apiInterceptor: HttpInterceptorFn = (request, next) => {
  const apiUrl = inject(API_URL);
  if (!request.url.startsWith(`${apiUrl}/`)) {
    return next(request);
  }

  const auth = inject(AuthService);
  const router = inject(Router);
  const notify = inject(Notify);

  return from(auth.idToken()).pipe(
    switchMap((token) =>
      next(token ? request.clone({ setHeaders: { Authorization: `Bearer ${token}` } }) : request),
    ),
    catchError((err: unknown) => {
      const error = ApiError.from(err);

      if (error.status === 401) {
        error.handled = true;
        const returnUrl = router.url;
        void auth.logout().then(() => {
          notify.error(SESSION_MESSAGES[error.message] ?? 'Please sign in to continue.');
          void router.navigate(['/login'], { queryParams: { returnUrl } });
        });
      } else if (error.status === 403 && error.message === 'email not verified') {
        error.handled = true;
        void router.navigate(['/verify-email']);
      } else if (error.status === 429) {
        error.handled = true;
        const wait = error.retryAfterSeconds
          ? ` Try again in ${error.retryAfterSeconds} seconds.`
          : '';
        notify.error(`Too many requests.${wait}`);
      }

      return throwError(() => error);
    }),
  );
};
