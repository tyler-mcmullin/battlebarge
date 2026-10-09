import { Component, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterLink } from '@angular/router';
import { LIMITS } from '../../api/limits';
import { AuthService } from '../../core/auth.service';
import { describeError } from '../../core/errors';
import { errorMessage } from '../../shared/form-errors';
import { notBlank, safeText } from '../../shared/validators';

@Component({
  selector: 'app-register',
  imports: [
    ReactiveFormsModule,
    RouterLink,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './register.html',
})
export class Register {
  private readonly fb = inject(NonNullableFormBuilder);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected readonly limits = LIMITS;
  protected readonly form = this.fb.group({
    email: ['', [Validators.required, Validators.email, Validators.maxLength(LIMITS.email)]],
    username: [
      '',
      [Validators.required, notBlank, safeText, Validators.maxLength(LIMITS.username)],
    ],
    password: [
      '',
      [
        Validators.required,
        Validators.minLength(LIMITS.passwordMin),
        Validators.maxLength(LIMITS.passwordMax),
      ],
    ],
  });
  protected readonly busy = signal(false);
  protected readonly error = signal('');
  protected readonly message = errorMessage;

  protected async submit(): Promise<void> {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }
    this.busy.set(true);
    this.error.set('');
    try {
      await this.auth.register(this.form.getRawValue());
      await this.router.navigate(['/verify-email']);
    } catch (err) {
      this.error.set(describeError(err));
    } finally {
      this.busy.set(false);
    }
  }
}
