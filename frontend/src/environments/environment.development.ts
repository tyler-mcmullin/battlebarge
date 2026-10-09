import { Environment } from './environment.model';

/**
 * Local development: talks to the Go API through the dev-server proxy and to the
 * Firebase Auth emulator (`firebase emulators:start`), so no real keys are needed.
 * The project id must match the one the emulator and the backend use.
 */
export const environment: Environment = {
  apiUrl: '/api',
  firebase: {
    apiKey: 'demo-api-key',
    authDomain: 'battlebarge-75869.firebaseapp.com',
    projectId: 'battlebarge-75869',
    appId: 'demo-app-id',
  },
  authEmulatorUrl: 'http://localhost:9099',
};
