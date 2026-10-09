import { HttpErrorResponse } from '@angular/common/http';
import { ApiError } from '../api/api-error';
import { describeError, isHandled } from './errors';

describe('describeError', () => {
  it("capitalizes the API's lower case messages", () => {
    expect(describeError(new ApiError('username already taken', 409))).toBe(
      'Username already taken',
    );
  });

  it("reads the API's message even from a raw HTTP error", () => {
    const raw = new HttpErrorResponse({ status: 409, error: { error: 'email already exists' } });
    expect(describeError(raw)).toBe('Email already exists');
  });

  it.each([
    ['auth/invalid-credential', 'Incorrect email or password.'],
    ['auth/wrong-password', 'Incorrect email or password.'],
    ['auth/user-not-found', 'Incorrect email or password.'],
    ['auth/user-disabled', 'This account has been disabled.'],
    ['auth/too-many-requests', 'Too many attempts. Wait a little and try again.'],
    [
      'auth/network-request-failed',
      'Cannot reach the server. Check your connection and try again.',
    ],
  ])('explains Firebase error %s', (code, expected) => {
    expect(describeError({ code })).toBe(expected);
  });

  it('does not reveal whether the email or the password was wrong', () => {
    expect(describeError({ code: 'auth/user-not-found' })).toBe(
      describeError({ code: 'auth/wrong-password' }),
    );
  });

  it('uses the message of unknown errors, with a generic fallback', () => {
    expect(describeError(new Error('custom'))).toBe('custom');
    expect(describeError(42)).toMatch(/something went wrong/i);
  });
});

describe('isHandled', () => {
  it('is true only for ApiErrors the interceptor already reported', () => {
    const error = new ApiError('x', 429);
    expect(isHandled(error)).toBe(false);
    error.handled = true;
    expect(isHandled(error)).toBe(true);
    expect(isHandled(new Error('x'))).toBe(false);
  });
});
