import { Component, DestroyRef, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router } from '@angular/router';
import { environment } from '../../../environments/environment';
import { AuthService } from '../../core/auth.service';
import { describeError } from '../../core/errors';
import { Notify } from '../../core/notify';

const RESEND_COOLDOWN_SECONDS = 30;

/**
 * Shown to signed-in users whose email is not verified yet. Firebase emails the
 * link; once it has been clicked, "I've verified" refreshes the account and the
 * token so the API accepts the user.
 */
@Component({
  selector: 'app-verify-email',
  imports: [MatCardModule, MatButtonModule, MatProgressSpinnerModule],
  templateUrl: './verify-email.html',
})
export class VerifyEmail {
  protected readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly notify = inject(Notify);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly busy = signal(false);
  protected readonly resendIn = signal(0);

  /** The emulator does not send real email; this page lists its verification links. */
  protected readonly emulatorLinksUrl = environment.authEmulatorUrl
    ? `${environment.authEmulatorUrl}/emulator/v1/projects/${environment.firebase.projectId}/oobCodes`
    : null;

  private timer: ReturnType<typeof setInterval> | undefined;

  constructor() {
    this.destroyRef.onDestroy(() => clearInterval(this.timer));
  }

  protected async continue(): Promise<void> {
    this.busy.set(true);
    try {
      if (await this.auth.refreshVerification()) {
        await this.router.navigate(['/warbands']);
      } else {
        this.notify.info('Not verified yet. Click the link in the email, then try again.');
      }
    } catch (err) {
      this.notify.error(describeError(err));
    } finally {
      this.busy.set(false);
    }
  }

  protected async resend(): Promise<void> {
    try {
      await this.auth.sendVerificationEmail();
      this.notify.info('Verification email sent.');
      this.startCooldown();
    } catch (err) {
      this.notify.error(describeError(err));
    }
  }

  protected async signOut(): Promise<void> {
    await this.auth.logout();
    await this.router.navigate(['/login']);
  }

  private startCooldown(): void {
    clearInterval(this.timer);
    this.resendIn.set(RESEND_COOLDOWN_SECONDS);
    this.timer = setInterval(() => {
      this.resendIn.update((seconds) => seconds - 1);
      if (this.resendIn() <= 0) {
        clearInterval(this.timer);
      }
    }, 1000);
  }
}
