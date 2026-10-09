import { HttpErrorResponse } from '@angular/common/http';

/**
 * An error from the API (or the network), reduced to what the UI needs. The Go
 * API always answers errors as `{"error": "<message>"}`.
 */
export class ApiError extends Error {
  /** Set by the HTTP interceptor once it has already told the user (toast or redirect). */
  handled = false;

  constructor(
    message: string,
    readonly status: number,
    /** Seconds to wait before retrying, from a 429's Retry-After header. */
    readonly retryAfterSeconds?: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }

  /** Builds an ApiError from whatever HttpClient gave us. */
  static from(err: unknown): ApiError {
    if (err instanceof ApiError) {
      return err;
    }
    if (err instanceof HttpErrorResponse) {
      if (err.status === 0) {
        return new ApiError('Cannot reach the server. Check your connection and try again.', 0);
      }
      const body = err.error as { error?: unknown } | null;
      const message =
        body && typeof body.error === 'string' ? body.error : `Request failed (${err.status})`;
      const retry = Number(err.headers?.get('Retry-After'));
      return new ApiError(
        message,
        err.status,
        Number.isFinite(retry) && retry > 0 ? retry : undefined,
      );
    }
    return new ApiError(err instanceof Error ? err.message : 'Something went wrong', -1);
  }
}
