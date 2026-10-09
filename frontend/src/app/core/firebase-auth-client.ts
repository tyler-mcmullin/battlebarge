import { Injectable } from '@angular/core';
import { FirebaseApp, initializeApp } from 'firebase/app';
import {
  Auth,
  connectAuthEmulator,
  getAuth,
  getIdToken,
  onAuthStateChanged,
  reload,
  sendEmailVerification,
  signInWithEmailAndPassword,
  signOut,
  User,
} from 'firebase/auth';
import { environment } from '../../environments/environment';
import { AuthClient, AuthUser } from './auth-client';

function toAuthUser(user: User): AuthUser {
  return { uid: user.uid, email: user.email, emailVerified: user.emailVerified };
}

/** The real AuthClient, backed by the Firebase web SDK. */
@Injectable()
export class FirebaseAuthClient extends AuthClient {
  private readonly app: FirebaseApp;
  private readonly auth: Auth;

  constructor() {
    super();
    if (!environment.firebase.apiKey || !environment.firebase.projectId) {
      throw new Error(
        'Firebase is not configured. Fill in `firebase` in src/environments/environment.ts.',
      );
    }
    this.app = initializeApp(environment.firebase);
    this.auth = getAuth(this.app);
    if (environment.authEmulatorUrl) {
      connectAuthEmulator(this.auth, environment.authEmulatorUrl, { disableWarnings: true });
    }
  }

  onChange(callback: (user: AuthUser | null) => void): void {
    onAuthStateChanged(this.auth, (user) => callback(user ? toAuthUser(user) : null));
  }

  async signIn(email: string, password: string): Promise<AuthUser> {
    const credential = await signInWithEmailAndPassword(this.auth, email, password);
    return toAuthUser(credential.user);
  }

  signOut(): Promise<void> {
    return signOut(this.auth);
  }

  async sendVerificationEmail(): Promise<void> {
    const user = this.auth.currentUser;
    if (!user) {
      throw new Error('Not signed in');
    }
    await sendEmailVerification(user);
  }

  async refresh(): Promise<AuthUser | null> {
    const user = this.auth.currentUser;
    if (!user) {
      return null;
    }
    await reload(user);
    await getIdToken(user, true);
    return toAuthUser(user);
  }

  async idToken(): Promise<string | null> {
    const user = this.auth.currentUser;
    return user ? getIdToken(user) : null;
  }
}
