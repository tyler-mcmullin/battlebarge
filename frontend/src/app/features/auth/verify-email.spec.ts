import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { FakeAuthClient } from '../../../testing/fake-auth-client';
import { API_URL } from '../../api/api-url';
import { AuthClient } from '../../core/auth-client';
import { Notify } from '../../core/notify';
import { VerifyEmail } from './verify-email';

describe('VerifyEmail', () => {
  let client: FakeAuthClient;
  let navigate: ReturnType<typeof vi.spyOn>;
  const notify = { info: vi.fn(), error: vi.fn() };

  async function setup() {
    notify.info.mockReset();
    notify.error.mockReset();
    client = new FakeAuthClient();
    client.user = { uid: 'u1', email: 'a@example.com', emailVerified: false };
    TestBed.configureTestingModule({
      imports: [VerifyEmail],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: AuthClient, useValue: client },
        { provide: Notify, useValue: notify },
      ],
    });
    navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    const fixture = TestBed.createComponent(VerifyEmail);
    await fixture.whenStable();
    fixture.detectChanges();
    return fixture;
  }

  const button = (html: HTMLElement, text: string) =>
    [...html.querySelectorAll('button')].find((b) => b.textContent?.includes(text));

  it('says which address the link was sent to', async () => {
    const fixture = await setup();
    expect((fixture.nativeElement as HTMLElement).textContent).toContain('a@example.com');
  });

  it('stays put and explains when the email is not verified yet', async () => {
    const fixture = await setup();
    button(fixture.nativeElement, "I've verified")?.click();
    await vi.waitFor(() => expect(notify.info).toHaveBeenCalled());
    expect(notify.info.mock.calls[0][0]).toMatch(/not verified yet/i);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('continues to the app once the email is verified', async () => {
    const fixture = await setup();
    client.refreshResult = { uid: 'u1', email: 'a@example.com', emailVerified: true };
    button(fixture.nativeElement, "I've verified")?.click();
    await vi.waitFor(() => expect(navigate).toHaveBeenCalledWith(['/warbands']));
    expect(client.calls).toContain('refresh');
  });

  it('resends the email, then makes the user wait before sending another', async () => {
    const fixture = await setup();
    button(fixture.nativeElement, 'Resend email')?.click();
    await vi.waitFor(() => expect(client.calls).toContain('sendVerificationEmail'));
    fixture.detectChanges();

    const resend = button(fixture.nativeElement, 'Resend in') as HTMLButtonElement | undefined;
    expect(resend?.disabled).toBe(true);
    expect(notify.info).toHaveBeenCalledWith('Verification email sent.');
  });

  it('signs out and returns to login', async () => {
    const fixture = await setup();
    button(fixture.nativeElement, 'Sign out')?.click();
    await vi.waitFor(() => expect(navigate).toHaveBeenCalledWith(['/login']));
    expect(client.calls).toContain('signOut');
  });
});
