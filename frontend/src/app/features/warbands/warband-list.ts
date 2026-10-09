import { Component, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Warband } from '../../api/models';
import { WarbandsApi } from '../../api/warbands.api';
import { describeError, isHandled } from '../../core/errors';
import { WarbandFormDialog } from './warband-form-dialog';

/** The signed-in user's warbands, with a button to create another. */
@Component({
  selector: 'app-warband-list',
  imports: [RouterLink, MatCardModule, MatButtonModule, MatIconModule, MatProgressSpinnerModule],
  templateUrl: './warband-list.html',
  styleUrl: './warband-list.scss',
})
export class WarbandList {
  private readonly api = inject(WarbandsApi);
  private readonly dialog = inject(MatDialog);
  private readonly router = inject(Router);

  protected readonly warbands = signal<Warband[]>([]);
  protected readonly loading = signal(true);
  protected readonly error = signal('');

  constructor() {
    void this.load();
  }

  protected async load(): Promise<void> {
    this.loading.set(true);
    this.error.set('');
    try {
      this.warbands.set(await firstValueFrom(this.api.list()));
    } catch (err) {
      if (!isHandled(err)) {
        this.error.set(describeError(err));
      }
    } finally {
      this.loading.set(false);
    }
  }

  protected async create(): Promise<void> {
    const created = await firstValueFrom(
      this.dialog
        .open<WarbandFormDialog, null, Warband>(WarbandFormDialog, { data: null })
        .afterClosed(),
    );
    if (created) {
      await this.router.navigate(['/warbands', created.id]);
    }
  }
}
