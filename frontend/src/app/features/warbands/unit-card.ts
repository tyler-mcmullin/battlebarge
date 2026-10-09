import { Component, computed, inject, input, linkedSignal, output, signal } from '@angular/core';
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

  /** Kills added (or removed, if negative) on screen but not yet sent to the API. */
  protected readonly pendingKills = signal(0);
  /** The kill count to show: the saved count plus the unsaved change. */
  protected readonly shownKills = computed(() => this.current().kills + this.pendingKills());

  /** Changes the on-screen kill count only; nothing is sent until `submitKills`. */
  protected adjustKills(amount: number): void {
    this.pendingKills.update((pending) => Math.max(pending + amount, -this.current().kills));
  }

  /** Sends the unsaved kill change to the API. On failure, it is kept so the user can retry. */
  protected async submitKills(): Promise<void> {
    const amount = this.pendingKills();
    if (amount === 0) {
      return;
    }
    if (await this.save(this.api.addKills(this.current().id, amount))) {
      this.pendingKills.set(0);
    }
  }

  /** XP added (or removed, if negative) on screen but not yet sent to the API. */
  protected readonly pendingXp = signal(0);
  /** The XP count to show: the saved count plus the unsaved change. */
  protected readonly shownXp = computed(() => this.current().experience + this.pendingXp());

  /** Changes the on-screen XP count only; nothing is sent until `submitXp`. */
  protected adjustXp(amount: number): void {
    this.pendingXp.update((pending) => Math.max(pending + amount, -this.current().experience));
  }

  /** Sends the unsaved XP change to the API. On failure, it is kept so the user can retry. */
  protected async submitXp(): Promise<void> {
    const amount = this.pendingXp();
    if (amount === 0) {
      return;
    }
    if (await this.save(this.api.addXp(this.current().id, amount))) {
      this.pendingXp.set(0);
    }
  }

  protected async removePerk(perk: Perk): Promise<void> {
    await this.save(this.api.deletePerk(this.current().id, perk.id));
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

  /** Runs one API call that returns the updated unit; resolves true if it succeeded. */
  private async save(request: Observable<Unit>): Promise<boolean> {
    this.busy.set(true);
    try {
      this.apply(await firstValueFrom(request));
      return true;
    } catch (err) {
      this.report(err);
      return false;
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
