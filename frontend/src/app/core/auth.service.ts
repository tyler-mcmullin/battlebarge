import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthApi } from '../api/auth.api';
import { RegisterRequest } from '../api/models';
import { AuthClient, AuthUser } from './auth-client';

/** Who is signed in, and the actions that change that. */
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly client = inject(AuthClient);
  private readonly api = inject(AuthApi);

  private readonly current = signal<AuthUser | null>(null);

  readonly user = this.current.asReadonly();
  readonly isSignedIn = computed(() => this.current() !== null);
  readonly emailVerified = computed(() => this.current()?.emailVerified ?? false);

  private readonly ready: Promise<void>;

  constructor() {
    // Firebase restores a saved session asynchronously, so the first callback is
    // what tells us whether someone is already signed in.
    this.ready = new Promise((resolve) => {
      this.client.onChange((user) => {
        this.current.set(user);
        resolve();
      });
    });
  }

  /** Resolves once the saved session (if any) has been restored; guards wait on this. */
  whenReady(): Promise<void> {
    return this.ready;
  }

  /**
   * Creates the account, signs in, and sends the verification email. The API
   * will refuse the new user until they have clicked the link in that email.
   */
  async register(request: RegisterRequest): Promise<void> {
    await firstValueFrom(this.api.register(request));
    await this.login(request.email, request.password);
    await this.client.sendVerificationEmail();
  }

  async login(email: string, password: string): Promise<void> {
    this.current.set(await this.client.signIn(email, password));
  }

  async logout(): Promise<void> {
    await this.client.signOut();
    this.current.set(null);
  }

  sendVerificationEmail(): Promise<void> {
    return this.client.sendVerificationEmail();
  }

  /** Re-checks verification with Firebase. Returns true once the email is verified. */
  async refreshVerification(): Promise<boolean> {
    const user = await this.client.refresh();
    this.current.set(user);
    return user?.emailVerified ?? false;
  }

  idToken(): Promise<string | null> {
    return this.client.idToken();
  }
}
