import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ActivatedRoute, provideRouter, Router } from '@angular/router';
import { FakeAuthClient } from '../../../testing/fake-auth-client';
import { submitForm, typeInto } from '../../../testing/dom';
import { API_URL } from '../../api/api-url';
import { AuthClient } from '../../core/auth-client';
import { Login } from './login';

describe('Login', () => {
  let client: FakeAuthClient;
  let navigateByUrl: ReturnType<typeof vi.spyOn>;

  function setup(returnUrl: string | null = null) {
    client = new FakeAuthClient();
    TestBed.configureTestingModule({
      imports: [Login],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: AuthClient, useValue: client },
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { queryParamMap: { get: () => returnUrl } } },
        },
      ],
    });
    navigateByUrl = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
    const fixture = TestBed.createComponent(Login);
    fixture.detectChanges();
    return fixture;
  }

  it('does not sign in with an empty or invalid form, and says what is wrong', async () => {
    const fixture = setup();
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[type=email]', 'not-an-email');
    submitForm(html);
    await fixture.whenStable();
    fixture.detectChanges();

    expect(client.calls).toEqual([]);
    expect(html.textContent).toContain('Enter a valid email address');
    expect(html.textContent).toContain('Required');
  });

  it('signs in and goes to the warbands page', async () => {
    const fixture = setup();
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[type=email]', 'a@example.com');
    typeInto(html, 'input[type=password]', 'secret123');
    submitForm(html);

    await vi.waitFor(() => expect(navigateByUrl).toHaveBeenCalledWith('/warbands'));
    expect(client.calls).toEqual(['signIn:a@example.com']);
  });

  it('returns to the page the user was trying to reach', async () => {
    const fixture = setup('/warbands/abc');
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[type=email]', 'a@example.com');
    typeInto(html, 'input[type=password]', 'secret123');
    submitForm(html);
    await vi.waitFor(() => expect(navigateByUrl).toHaveBeenCalledWith('/warbands/abc'));
  });

  it('ignores a return URL that points to another site', async () => {
    const fixture = setup('https://evil.example/steal');
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[type=email]', 'a@example.com');
    typeInto(html, 'input[type=password]', 'secret123');
    submitForm(html);
    await vi.waitFor(() => expect(navigateByUrl).toHaveBeenCalledWith('/warbands'));
  });

  it('sends an unverified user to the verification page', async () => {
    const fixture = setup('/warbands/abc');
    client.signInResult = { uid: 'u1', email: 'a@example.com', emailVerified: false };
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[type=email]', 'a@example.com');
    typeInto(html, 'input[type=password]', 'secret123');
    submitForm(html);
    await vi.waitFor(() => expect(navigateByUrl).toHaveBeenCalledWith('/verify-email'));
  });

  it('shows a friendly message when the credentials are wrong', async () => {
    const fixture = setup();
    client.signInError = Object.assign(new Error('x'), { code: 'auth/invalid-credential' });
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[type=email]', 'a@example.com');
    typeInto(html, 'input[type=password]', 'wrong-password');
    submitForm(html);

    await vi.waitFor(() => {
      fixture.detectChanges();
      expect(html.querySelector('[role=alert]')?.textContent).toContain(
        'Incorrect email or password.',
      );
    });
    expect(navigateByUrl).not.toHaveBeenCalled();
  });
});
