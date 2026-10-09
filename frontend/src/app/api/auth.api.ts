import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { API_URL } from './api-url';
import { RegisterRequest, RegisterResponse, User } from './models';

/** Calls for the API's own account endpoints (sign-in itself is done with Firebase). */
@Injectable({ providedIn: 'root' })
export class AuthApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_URL);

  /** Creates the account. Afterwards the user signs in with Firebase using the same email and password. */
  register(body: RegisterRequest) {
    return this.http.post<RegisterResponse>(`${this.base}/auth/register`, body);
  }

  me() {
    return this.http.get<User>(`${this.base}/users/me`);
  }
}
