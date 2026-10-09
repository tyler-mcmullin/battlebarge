import { HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { ApiError } from './api-error';

describe('ApiError.from', () => {
  it("reads the API's {error} message and status", () => {
    const err = ApiError.from(
      new HttpErrorResponse({ status: 409, error: { error: 'username already taken' } }),
    );
    expect(err.message).toBe('username already taken');
    expect(err.status).toBe(409);
    expect(err.handled).toBe(false);
  });

  it('falls back to the status code when the body is not the usual shape', () => {
    expect(ApiError.from(new HttpErrorResponse({ status: 500, error: 'boom' })).message).toBe(
      'Request failed (500)',
    );
    expect(ApiError.from(new HttpErrorResponse({ status: 502, error: null })).message).toBe(
      'Request failed (502)',
    );
  });

  it('reports an unreachable server (status 0) in plain words', () => {
    const err = ApiError.from(
      new HttpErrorResponse({ status: 0, error: new ProgressEvent('error') }),
    );
    expect(err.status).toBe(0);
    expect(err.message).toMatch(/cannot reach the server/i);
  });

  it('reads Retry-After from a 429', () => {
    const err = ApiError.from(
      new HttpErrorResponse({
        status: 429,
        error: { error: 'too many requests' },
        headers: new HttpHeaders({ 'Retry-After': '42' }),
      }),
    );
    expect(err.retryAfterSeconds).toBe(42);
  });

  it('ignores a missing or invalid Retry-After', () => {
    expect(ApiError.from(new HttpErrorResponse({ status: 429 })).retryAfterSeconds).toBeUndefined();
    const bad = new HttpErrorResponse({
      status: 429,
      headers: new HttpHeaders({ 'Retry-After': 'soon' }),
    });
    expect(ApiError.from(bad).retryAfterSeconds).toBeUndefined();
  });

  it('passes an existing ApiError through unchanged', () => {
    const original = new ApiError('x', 400);
    expect(ApiError.from(original)).toBe(original);
  });

  it('wraps other errors', () => {
    expect(ApiError.from(new Error('oops')).message).toBe('oops');
    expect(ApiError.from('weird').message).toBe('Something went wrong');
  });
});
