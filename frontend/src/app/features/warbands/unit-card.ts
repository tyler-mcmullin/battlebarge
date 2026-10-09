import { Component, inject, input, linkedSignal, output, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { firstValueFrom, Observable } from 'rxjs';
import { Perk, Unit } from '../../api/models';
import { UnitsApi } from '../../api/units.api';
import { describeError, isHandled } from '../../core/errors';
import { Notify } from '../../core/notify';
import { ConfirmDialog } from '../../shared/confirm-dialog';
import { PerkDialog } from './perk-dialog';
import { UnitFormData, UnitFormDialog } from './unit-form-dialog';

/**
 * One unit: its kills and experience (with +/- buttons), its perks and scars, and
 * edit/delete. It keeps its own copy of the unit so the numbers update instantly,
 * and tells the page (`changed`) so warband totals can be refreshed.
 */
@Component({
  selector: 'app-unit-card',
  imports: [MatCardModule, MatButtonModule, MatIconModule, MatMenuModule],
  templateUrl: './unit-card.html',
  styleUrl: './unit-card.scss',
})
export class UnitCard {
  private readonly api = inject(UnitsApi);
  private readonly dialog = inject(MatDialog);
  private readonly notify = inject(Notify);

  readonly unit = input.required<Unit>();
  readonly changed = output<void>();
  readonly deleted = output<void>();

  /** Starts as the unit from the page and is replaced by each response from the API. */
  protected readonly current = linkedSignal(() => this.unit());
  protected readonly busy = signal(false);

  protected adjustKills(amount: number): Promise<void> {
    return this.save(this.api.addKills(this.current().id, amount));
  }

  protected adjustXp(amount: number): Promise<void> {
    return this.save(this.api.addXp(this.current().id, amount));
  }

  protected removePerk(perk: Perk): Promise<void> {
    return this.save(this.api.deletePerk(this.current().id, perk.id));
  }

  protected async addPerk(): Promise<void> {
    const updated = await firstValueFrom(
      this.dialog
        .open<PerkDialog, string, Unit>(PerkDialog, { data: this.current().id })
        .afterClosed(),
    );
    this.apply(updated);
  }

  protected async edit(): Promise<void> {
    const updated = await firstValueFrom(
      this.dialog
        .open<UnitFormDialog, UnitFormData, Unit>(UnitFormDialog, {
          data: { warbandId: this.current().warband_id, unit: this.current() },
        })
        .afterClosed(),
    );
    this.apply(updated);
  }

  protected async remove(): Promise<void> {
    const confirmed = await firstValueFrom(
      this.dialog
        .open(ConfirmDialog, {
          data: {
            title: 'Delete unit?',
            message: `${this.current().unit_name} and its perks will be deleted.`,
            confirmLabel: 'Delete',
          },
        })
        .afterClosed(),
    );
    if (!confirmed) {
      return;
    }
    this.busy.set(true);
    try {
      await firstValueFrom(this.api.delete(this.current().id));
      this.deleted.emit();
    } catch (err) {
      this.report(err);
    } finally {
      this.busy.set(false);
    }
  }

  private apply(updated: Unit | undefined): void {
    if (updated) {
      this.current.set(updated);
      this.changed.emit();
    }
  }

  /** Runs one API call that returns the updated unit. */
  private async save(request: Observable<Unit>): Promise<void> {
    this.busy.set(true);
    try {
      this.apply(await firstValueFrom(request));
    } catch (err) {
      this.report(err);
    } finally {
      this.busy.set(false);
    }
  }

  private report(err: unknown): void {
    if (!isHandled(err)) {
      this.notify.error(describeError(err));
    }
  }
}
