import { HttpErrorResponse } from '@angular/common/http';
import { ApiError } from '../api/api-error';

function capitalize(text: string): string {
  return text ? text.charAt(0).toUpperCase() + text.slice(1) : text;
}

/** Turns an error from Firebase or the API into a sentence a person can act on. */
export function describeError(err: unknown): string {
  if (err instanceof ApiError || err instanceof HttpErrorResponse) {
    // the API's messages are lower case, e.g. "username already taken"
    return capitalize(ApiError.from(err).message);
  }
  const code =
    typeof err === 'object' && err !== null ? (err as { code?: unknown }).code : undefined;
  switch (code) {
    case 'auth/invalid-credential':
    case 'auth/wrong-password':
    case 'auth/user-not-found':
    case 'auth/invalid-email':
      return 'Incorrect email or password.';
    case 'auth/user-disabled':
      return 'This account has been disabled.';
    case 'auth/too-many-requests':
      return 'Too many attempts. Wait a little and try again.';
    case 'auth/network-request-failed':
      return 'Cannot reach the server. Check your connection and try again.';
    default:
      return err instanceof Error ? err.message : 'Something went wrong. Please try again.';
  }
}

/** True when the HTTP interceptor already told the user about this error (toast or redirect). */
export function isHandled(err: unknown): boolean {
  return err instanceof ApiError && err.handled;
}
