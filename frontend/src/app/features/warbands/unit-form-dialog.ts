import { Component, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { firstValueFrom } from 'rxjs';
import { LIMITS } from '../../api/limits';
import { Unit } from '../../api/models';
import { UnitsApi } from '../../api/units.api';
import { describeError, isHandled } from '../../core/errors';
import { errorMessage } from '../../shared/form-errors';
import { notBlank, safeText, wholeNumber } from '../../shared/validators';

/** What the dialog is given: the warband to add to, and optionally the unit to edit. */
export interface UnitFormData {
  warbandId: string;
  unit: Unit | null;
}

/** Adds a unit to a warband, or edits one. Closes with the saved unit. */
@Component({
  selector: 'app-unit-form-dialog',
  imports: [
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './unit-form-dialog.html',
})
export class UnitFormDialog {
  private readonly fb = inject(NonNullableFormBuilder);
  private readonly api = inject(UnitsApi);
  private readonly ref = inject<MatDialogRef<UnitFormDialog, Unit>>(MatDialogRef);
  protected readonly data = inject<UnitFormData>(MAT_DIALOG_DATA);

  protected readonly limits = LIMITS;
  protected readonly form = this.fb.group({
    unit_name: [
      this.data.unit?.unit_name ?? '',
      [Validators.required, notBlank, safeText, Validators.maxLength(LIMITS.name)],
    ],
    narrative_name: [
      this.data.unit?.narrative_name ?? '',
      [safeText, Validators.maxLength(LIMITS.name)],
    ],
    bio: [this.data.unit?.bio ?? '', [safeText, Validators.maxLength(LIMITS.unitBio)]],
    points: [
      this.data.unit?.points ?? 0,
      [
        Validators.required,
        Validators.min(LIMITS.points.min),
        Validators.max(LIMITS.points.max),
        wholeNumber,
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
    const value = this.form.getRawValue();
    const body = { ...value, points: Number(value.points) };
    try {
      const saved = await firstValueFrom(
        this.data.unit
          ? this.api.update(this.data.unit.id, body)
          : this.api.create({ warband_id: this.data.warbandId, ...body }),
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
