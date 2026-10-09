import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter, withComponentInputBinding } from '@angular/router';

import { routes } from './app.routes';
import { AuthClient } from './core/auth-client';
import { apiInterceptor } from './core/api.interceptor';
import { FirebaseAuthClient } from './core/firebase-auth-client';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    // withComponentInputBinding lets a route's :id parameter arrive as a component input
    provideRouter(routes, withComponentInputBinding()),
    provideHttpClient(withInterceptors([apiInterceptor])),
    { provide: AuthClient, useClass: FirebaseAuthClient },
  ],
};
