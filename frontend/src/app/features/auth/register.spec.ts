import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { FakeAuthClient } from '../../../testing/fake-auth-client';
import { submitForm, typeInto } from '../../../testing/dom';
import { API_URL } from '../../api/api-url';
import { AuthClient } from '../../core/auth-client';
import { Register } from './register';

describe('Register', () => {
  let client: FakeAuthClient;
  let http: HttpTestingController;
  let navigate: ReturnType<typeof vi.spyOn>;

  function setup() {
    client = new FakeAuthClient();
    client.signInResult = { uid: 'u1', email: 'a@example.com', emailVerified: false };
    TestBed.configureTestingModule({
      imports: [Register],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: AuthClient, useValue: client },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    const fixture = TestBed.createComponent(Register);
    fixture.detectChanges();
    return fixture;
  }

  function fill(
    html: HTMLElement,
    overrides: Partial<Record<'email' | 'username' | 'password', string>> = {},
  ) {
    typeInto(html, 'input[type=email]', overrides.email ?? 'a@example.com');
    typeInto(html, 'input[formControlName=username]', overrides.username ?? 'alice');
    typeInto(html, 'input[type=password]', overrides.password ?? 'secret123');
  }

  it('creates the account, then sends the user to verify their email', async () => {
    const fixture = setup();
    const html = fixture.nativeElement as HTMLElement;
    fill(html);
    submitForm(html);

    const req = http.expectOne('/api/auth/register');
    expect(req.request.body).toEqual({
      email: 'a@example.com',
      username: 'alice',
      password: 'secret123',
    });
    req.flush(
      { message: 'user created', user_id: 'u1', email_verification_required: true },
      { status: 201, statusText: 'Created' },
    );

    await vi.waitFor(() => expect(navigate).toHaveBeenCalledWith(['/verify-email']));
    expect(client.calls).toEqual(['signIn:a@example.com', 'sendVerificationEmail']);
  });

  it.each([
    ['a blank username', { username: '   ' }, 'Cannot be only spaces'],
    ['a username over 50 characters', { username: 'u'.repeat(51) }, 'At most 50 characters'],
    ['a short password', { password: 'short' }, 'At least 8 characters'],
    ['an invalid email', { email: 'nope' }, 'Enter a valid email address'],
  ])('rejects %s before calling the API', async (_label, overrides, message) => {
    const fixture = setup();
    const html = fixture.nativeElement as HTMLElement;
    fill(html, overrides);
    submitForm(html);
    await fixture.whenStable();
    fixture.detectChanges();

    http.expectNone('/api/auth/register');
    expect(html.textContent).toContain(message);
  });

  it("shows the API's message when the username is taken", async () => {
    const fixture = setup();
    const html = fixture.nativeElement as HTMLElement;
    fill(html);
    submitForm(html);
    http
      .expectOne('/api/auth/register')
      .flush({ error: 'username already taken' }, { status: 409, statusText: 'Conflict' });

    await vi.waitFor(() => {
      fixture.detectChanges();
      expect(html.querySelector('[role=alert]')?.textContent).toContain('Username already taken');
    });
    expect(navigate).not.toHaveBeenCalled();
    expect(client.calls).toEqual([]);
  });
});
