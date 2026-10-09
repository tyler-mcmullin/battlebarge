/** Settings that differ between local development and a deployed build. */
export interface Environment {
  /**
   * Where the Go API lives. In development this is "/api", which the dev
   * server proxies to http://localhost:8080 (see proxy.conf.json), so no CORS
   * setup is needed. In production, it is the API's real URL.
   */
  apiUrl: string;

  /** The Firebase web app config. These values are public by design, not secrets. */
  firebase: {
    apiKey: string;
    authDomain: string;
    projectId: string;
    appId: string;
  };

  /** URL of the Firebase Auth emulator, or null to use real Firebase. */
  authEmulatorUrl: string | null;
}
