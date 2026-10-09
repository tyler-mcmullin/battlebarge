import { AuthClient, AuthUser } from '../app/core/auth-client';

/** A stand-in for Firebase that tests can steer, recording what the app asked it to do. */
export class FakeAuthClient extends AuthClient {
  user: AuthUser | null = null;
  token = 'test-token';
  signInResult: AuthUser = { uid: 'u1', email: 'a@example.com', emailVerified: true };
  signInError: unknown = null;
  refreshResult: AuthUser | null | undefined = undefined;
  readonly calls: string[] = [];

  onChange(callback: (user: AuthUser | null) => void): void {
    callback(this.user);
  }

  async signIn(email: string): Promise<AuthUser> {
    this.calls.push(`signIn:${email}`);
    if (this.signInError) {
      throw this.signInError;
    }
    this.user = this.signInResult;
    return this.signInResult;
  }

  async signOut(): Promise<void> {
    this.calls.push('signOut');
    this.user = null;
  }

  async sendVerificationEmail(): Promise<void> {
    this.calls.push('sendVerificationEmail');
  }

  async refresh(): Promise<AuthUser | null> {
    this.calls.push('refresh');
    return this.refreshResult === undefined ? this.user : this.refreshResult;
  }

  async idToken(): Promise<string | null> {
    return this.user ? this.token : null;
  }
}
