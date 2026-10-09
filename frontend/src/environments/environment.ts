import { Environment } from './environment.model';

/**
 * Production build. Fill these in before deploying (see PRODUCTION_CHECKLIST.md):
 *  - apiUrl: the deployed API, e.g. "https://api.example.com" (or "/api" behind a
 *    reverse proxy). The API's CORS_ALLOWED_ORIGINS must include this site's origin
 *    when the API is on a different origin.
 *  - firebase: the web app config from the Firebase console (Project settings,
 *    Your apps). These identify the project; they are not secrets.
 */
export const environment: Environment = {
  apiUrl: '',
  firebase: {
    apiKey: '',
    authDomain: '',
    projectId: '',
    appId: '',
  },
  authEmulatorUrl: null,
};
