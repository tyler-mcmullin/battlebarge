/** The few things the app needs to know about the signed-in Firebase account. */
export interface AuthUser {
  uid: string;
  email: string | null;
  emailVerified: boolean;
}

/**
 * The app's view of Firebase Authentication. Components and services depend on
 * this, not on the Firebase library, so tests can swap in a fake and the
 * library stays in one file (firebase-auth-client.ts).
 */
export abstract class AuthClient {
  /** Calls back with the current user (or null) now and whenever it changes. */
  abstract onChange(callback: (user: AuthUser | null) => void): void;

  abstract signIn(email: string, password: string): Promise<AuthUser>;

  abstract signOut(): Promise<void>;

  /** Asks Firebase to email the signed-in user a verification link. */
  abstract sendVerificationEmail(): Promise<void>;

  /**
   * Re-reads the account from Firebase and gets a freshly issued ID token. Needed
   * after the user clicks the verification link, because the email_verified claim
   * the API checks is baked into the token when it is issued.
   */
  abstract refresh(): Promise<AuthUser | null>;

  /** The current ID token to send to the API, or null when signed out. */
  abstract idToken(): Promise<string | null>;
}
