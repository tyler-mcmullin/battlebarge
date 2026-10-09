import { Component, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { firstValueFrom } from 'rxjs';
import { LIMITS } from '../../api/limits';
import { Warband } from '../../api/models';
import { WarbandsApi } from '../../api/warbands.api';
import { describeError, isHandled } from '../../core/errors';
import { errorMessage } from '../../shared/form-errors';
import { notBlank, safeText, wholeNumber } from '../../shared/validators';

const pointsValidators = [
  Validators.required,
  Validators.min(LIMITS.points.min),
  Validators.max(LIMITS.points.max),
  wholeNumber,
];

/** Creates a warband, or edits the one passed as dialog data. Closes with the saved warband. */
@Component({
  selector: 'app-warband-form-dialog',
  imports: [
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './warband-form-dialog.html',
})
export class WarbandFormDialog {
  private readonly fb = inject(NonNullableFormBuilder);
  private readonly api = inject(WarbandsApi);
  private readonly ref = inject<MatDialogRef<WarbandFormDialog, Warband>>(MatDialogRef);
  protected readonly warband = inject<Warband | null>(MAT_DIALOG_DATA);

  protected readonly limits = LIMITS;
  protected readonly form = this.fb.group({
    name: [
      this.warband?.name ?? '',
      [Validators.required, notBlank, safeText, Validators.maxLength(LIMITS.name)],
    ],
    faction: [this.warband?.faction ?? '', [safeText, Validators.maxLength(LIMITS.faction)]],
    description: [
      this.warband?.description ?? '',
      [safeText, Validators.maxLength(LIMITS.warbandDescription)],
    ],
    requisition_points: [this.warband?.requisition_points ?? 0, pointsValidators],
    supply_limit: [this.warband?.supply_limit ?? 0, pointsValidators],
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
    const value = this.form.getRawValue();
    const body = {
      ...value,
      requisition_points: Number(value.requisition_points),
      supply_limit: Number(value.supply_limit),
    };
    try {
      const saved = await firstValueFrom(
        this.warband ? this.api.update(this.warband.id, body) : this.api.create(body),
      );
      this.ref.close(saved);
    } catch (err) {
      if (!isHandled(err)) {
        this.error.set(describeError(err));
      }
    } finally {
      this.busy.set(false);
    }
  }
}
