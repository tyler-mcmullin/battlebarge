import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material/dialog';
import { provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';
import { makeWarband } from '../../../testing/factories';
import { API_URL } from '../../api/api-url';
import { WarbandList } from './warband-list';

describe('WarbandList', () => {
  let http: HttpTestingController;

  function setup() {
    TestBed.configureTestingModule({
      imports: [WarbandList],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(WarbandList);
    fixture.detectChanges();
    return fixture;
  }

  afterEach(() => http.verify());

  it('shows a card for each warband with its totals', async () => {
    const fixture = setup();
    http.expectOne('/api/warbands').flush([
      makeWarband({
        id: 'a',
        name: 'Da Boyz',
        faction: 'Orks',
        num_units: 3,
        total_points_cost: 240,
        crusade_points: 4,
      }),
      makeWarband({ id: 'b', name: 'Iron Hands', faction: 'Space Marines' }),
    ]);
    await fixture.whenStable();
    fixture.detectChanges();

    const html = fixture.nativeElement as HTMLElement;
    const cards = html.querySelectorAll('mat-card');
    expect(cards.length).toBe(2);
    expect(cards[0].textContent).toContain('Da Boyz');
    expect(cards[0].textContent).toContain('Orks');
    expect(cards[0].textContent).toContain('240');
    expect(cards[0].querySelector('a')?.getAttribute('href')).toBe('/warbands/a');
  });

  it('invites the user to create their first warband when there are none', async () => {
    const fixture = setup();
    http.expectOne('/api/warbands').flush([]);
    await fixture.whenStable();
    fixture.detectChanges();
    expect((fixture.nativeElement as HTMLElement).textContent).toContain(
      "You don't have any warbands yet",
    );
  });

  it('shows the failure and lets the user retry', async () => {
    const fixture = setup();
    http
      .expectOne('/api/warbands')
      .flush({ error: 'internal server error' }, { status: 500, statusText: 'Server Error' });
    await fixture.whenStable();
    fixture.detectChanges();

    const html = fixture.nativeElement as HTMLElement;
    expect(html.querySelector('[role=alert]')?.textContent).toContain('Internal server error');

    const retry = [...html.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Try again'),
    );
    retry?.click();
    http.expectOne('/api/warbands').flush([makeWarband()]);
    await fixture.whenStable();
    fixture.detectChanges();
    expect(html.querySelectorAll('mat-card').length).toBe(1);
  });

  it('opens the new warband after creating one', async () => {
    const fixture = setup();
    http.expectOne('/api/warbands').flush([]);
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    vi.spyOn(TestBed.inject(MatDialog), 'open').mockReturnValue({
      afterClosed: () => of(makeWarband({ id: 'new-id' })),
    } as never);
    await fixture.whenStable();
    fixture.detectChanges();

    const create = [...(fixture.nativeElement as HTMLElement).querySelectorAll('button')].find(
      (b) => b.textContent?.includes('New warband'),
    );
    create?.click();
    await vi.waitFor(() => expect(navigate).toHaveBeenCalledWith(['/warbands', 'new-id']));
  });
});
