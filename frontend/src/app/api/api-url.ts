import { InjectionToken } from '@angular/core';
import { environment } from '../../environments/environment';

/** Base URL of the Go API. Tests (and other environments) can provide a different one. */
export const API_URL = new InjectionToken<string>('API_URL', {
  providedIn: 'root',
  factory: () => environment.apiUrl,
});
