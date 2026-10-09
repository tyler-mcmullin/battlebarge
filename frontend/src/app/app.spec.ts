import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { FakeAuthClient } from '../testing/fake-auth-client';
import { App } from './app';
import { API_URL } from './api/api-url';
import { AuthClient } from './core/auth-client';

describe('App shell', () => {
  function setup(client: FakeAuthClient) {
    TestBed.configureTestingModule({
      imports: [App],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: AuthClient, useValue: client },
      ],
    });
    return TestBed.createComponent(App);
  }

  it('shows the brand, and no sign-out button when signed out', async () => {
    const fixture = setup(new FakeAuthClient());
    await fixture.whenStable();
    const html = fixture.nativeElement as HTMLElement;
    expect(html.querySelector('.brand')?.textContent).toContain('Battlebarge');
    expect(html.textContent).not.toContain('Sign out');
  });

  it('shows who is signed in, and signs out', async () => {
    const client = new FakeAuthClient();
    client.user = { uid: 'u1', email: 'a@example.com', emailVerified: true };
    const fixture = setup(client);
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    await fixture.whenStable();
    const html = fixture.nativeElement as HTMLElement;
    expect(html.querySelector('.who')?.textContent).toContain('a@example.com');

    const button = [...html.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Sign out'),
    );
    button?.click();
    await vi.waitFor(() => expect(navigate).toHaveBeenCalledWith(['/login']));
    expect(client.calls).toContain('signOut');
  });
});
