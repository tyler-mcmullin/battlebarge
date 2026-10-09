import { Routes } from '@angular/router';
import { authGuard, guestGuard, verifiedGuard } from './core/guards';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'warbands' },
  {
    path: 'login',
    title: 'Sign in',
    canActivate: [guestGuard],
    loadComponent: () => import('./features/auth/login').then((m) => m.Login),
  },
  {
    path: 'register',
    title: 'Create account',
    canActivate: [guestGuard],
    loadComponent: () => import('./features/auth/register').then((m) => m.Register),
  },
  {
    path: 'verify-email',
    title: 'Verify your email',
    canActivate: [authGuard],
    loadComponent: () => import('./features/auth/verify-email').then((m) => m.VerifyEmail),
  },
  {
    path: 'warbands',
    title: 'Warbands',
    canActivate: [verifiedGuard],
    loadComponent: () => import('./features/warbands/warband-list').then((m) => m.WarbandList),
  },
  {
    path: 'warbands/:id',
    title: 'Warband',
    canActivate: [verifiedGuard],
    loadComponent: () => import('./features/warbands/warband-detail').then((m) => m.WarbandDetail),
  },
  { path: '**', redirectTo: 'warbands' },
];
