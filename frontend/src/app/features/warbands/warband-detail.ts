import { Component, effect, inject, input, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../api/api-error';
import { Unit, Warband } from '../../api/models';
import { WarbandsApi } from '../../api/warbands.api';
import { describeError, isHandled } from '../../core/errors';
import { Notify } from '../../core/notify';
import { ConfirmDialog } from '../../shared/confirm-dialog';
import { UnitCard } from './unit-card';
import { UnitFormData, UnitFormDialog } from './unit-form-dialog';
import { WarbandFormDialog } from './warband-form-dialog';

/** One warband with its totals and its units. */
@Component({
  selector: 'app-warband-detail',
  imports: [RouterLink, MatButtonModule, MatIconModule, MatProgressSpinnerModule, UnitCard],
  templateUrl: './warband-detail.html',
  styleUrl: './warband-detail.scss',
})
export class WarbandDetail {
  private readonly api = inject(WarbandsApi);
  private readonly dialog = inject(MatDialog);
  private readonly router = inject(Router);
  private readonly notify = inject(Notify);

  /** The :id from the URL (see withComponentInputBinding in app.config.ts). */
  readonly id = input.required<string>();

  protected readonly warband = signal<Warband | null>(null);
  protected readonly loading = signal(true);
  protected readonly error = signal('');
  protected readonly notFound = signal(false);

  constructor() {
    effect(() => {
      void this.load(this.id(), true);
    });
  }

  /** Fetches the warband. `showSpinner` is false for quiet refreshes after a unit changes. */
  protected async load(id: string, showSpinner: boolean): Promise<void> {
    if (showSpinner) {
      this.loading.set(true);
      this.error.set('');
      this.notFound.set(false);
    }
    try {
      this.warband.set(await firstValueFrom(this.api.get(id)));
    } catch (err) {
      if (ApiError.from(err).status === 404) {
        this.notFound.set(true);
      } else if (!isHandled(err)) {
        this.error.set(describeError(err));
      }
    } finally {
      this.loading.set(false);
    }
  }

  /** Re-reads the warband so its totals reflect a change to one of its units. */
  protected refresh(): Promise<void> {
    return this.load(this.id(), false);
  }

  protected async edit(warband: Warband): Promise<void> {
    const saved = await firstValueFrom(
      this.dialog
        .open<WarbandFormDialog, Warband, Warband>(WarbandFormDialog, { data: warband })
        .afterClosed(),
    );
    if (saved) {
      await this.refresh();
    }
  }

  protected async addUnit(warband: Warband): Promise<void> {
    const created = await firstValueFrom(
      this.dialog
        .open<UnitFormDialog, UnitFormData, Unit>(UnitFormDialog, {
          data: { warbandId: warband.id, unit: null },
        })
        .afterClosed(),
    );
    if (created) {
      await this.refresh();
    }
  }

  protected async remove(warband: Warband): Promise<void> {
    const confirmed = await firstValueFrom(
      this.dialog
        .open(ConfirmDialog, {
          data: {
            title: 'Delete warband?',
            message: `${warband.name} and all of its units will be deleted. This cannot be undone.`,
            confirmLabel: 'Delete',
          },
        })
        .afterClosed(),
    );
    if (!confirmed) {
      return;
    }
    try {
      await firstValueFrom(this.api.delete(warband.id));
      this.notify.info(`${warband.name} was deleted.`);
      await this.router.navigate(['/warbands']);
    } catch (err) {
      if (!isHandled(err)) {
        this.notify.error(describeError(err));
      }
    }
  }
}
