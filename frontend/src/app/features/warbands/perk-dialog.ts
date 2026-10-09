import { Component, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { firstValueFrom } from 'rxjs';
import { LIMITS } from '../../api/limits';
import { Unit } from '../../api/models';
import { UnitsApi } from '../../api/units.api';
import { describeError, isHandled } from '../../core/errors';
import { errorMessage } from '../../shared/form-errors';
import { notBlank, safeText } from '../../shared/validators';

/** Adds a perk (or a scar) to the unit whose id is passed as dialog data. Closes with the updated unit. */
@Component({
  selector: 'app-perk-dialog',
  imports: [
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSlideToggleModule,
    MatButtonModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './perk-dialog.html',
})
export class PerkDialog {
  private readonly fb = inject(NonNullableFormBuilder);
  private readonly api = inject(UnitsApi);
  private readonly ref = inject<MatDialogRef<PerkDialog, Unit>>(MatDialogRef);
  private readonly unitId = inject<string>(MAT_DIALOG_DATA);

  protected readonly limits = LIMITS;
  protected readonly form = this.fb.group({
    name: ['', [Validators.required, notBlank, safeText, Validators.maxLength(LIMITS.name)]],
    description: ['', [safeText, Validators.maxLength(LIMITS.perkDescription)]],
    is_scar: [false],
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
      this.ref.close(await firstValueFrom(this.api.addPerk(this.unitId, this.form.getRawValue())));
    } catch (err) {
      if (!isHandled(err)) {
        this.error.set(describeError(err));
      }
    } finally {
      this.busy.set(false);
    }
  }
}
