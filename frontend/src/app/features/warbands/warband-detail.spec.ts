import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material/dialog';
import { provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';
import { makeUnit, makeWarband } from '../../../testing/factories';
import { API_URL } from '../../api/api-url';
import { Notify } from '../../core/notify';
import { WarbandDetail } from './warband-detail';

describe('WarbandDetail', () => {
  let http: HttpTestingController;
  const notify = { info: vi.fn(), error: vi.fn() };

  function setup() {
    notify.info.mockReset();
    notify.error.mockReset();
    TestBed.configureTestingModule({
      imports: [WarbandDetail],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: Notify, useValue: notify },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(WarbandDetail);
    fixture.componentRef.setInput('id', 'wb-1');
    fixture.detectChanges();
    return fixture;
  }

  const html = (f: { nativeElement: unknown }) => f.nativeElement as HTMLElement;
  afterEach(() => http.verify());

  it('loads the warband named in the URL and shows its totals and units', async () => {
    const fixture = setup();
    http.expectOne('/api/warbands/wb-1').flush(
      makeWarband({
        name: 'Da Boyz',
        faction: 'Orks',
        num_units: 2,
        total_points_cost: 160,
        crusade_points: 3,
        units: [makeUnit({ id: 'a', unit_name: 'Boss' }), makeUnit({ id: 'b', unit_name: 'Nob' })],
      }),
    );
    await fixture.whenStable();
    fixture.detectChanges();

    expect(html(fixture).querySelector('h1')?.textContent).toContain('Da Boyz');
    expect(html(fixture).textContent).toContain('160');
    expect(html(fixture).querySelectorAll('app-unit-card').length).toBe(2);
  });

  it('says so when the warband does not exist or is not the user’s', async () => {
    const fixture = setup();
    http
      .expectOne('/api/warbands/wb-1')
      .flush({ error: 'warband not found' }, { status: 404, statusText: 'Not Found' });
    await fixture.whenStable();
    fixture.detectChanges();
    expect(html(fixture).textContent).toContain('does not exist');
  });

  it('shows an empty state when there are no units', async () => {
    const fixture = setup();
    http.expectOne('/api/warbands/wb-1').flush(makeWarband({ units: [] }));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(html(fixture).textContent).toContain('No units yet');
  });

  it('refreshes the totals after a unit is added', async () => {
    const fixture = setup();
    http.expectOne('/api/warbands/wb-1').flush(makeWarband({ units: [] }));
    vi.spyOn(TestBed.inject(MatDialog), 'open').mockReturnValue({
      afterClosed: () => of(makeUnit()),
    } as never);
    await fixture.whenStable();
    fixture.detectChanges();

    [...html(fixture).querySelectorAll('button')]
      .find((b) => b.textContent?.includes('Add unit'))
      ?.click();
    await vi.waitFor(() =>
      http
        .expectOne('/api/warbands/wb-1')
        .flush(makeWarband({ num_units: 1, units: [makeUnit()] })),
    );
    await fixture.whenStable();
    fixture.detectChanges();
    expect(html(fixture).querySelectorAll('app-unit-card').length).toBe(1);
  });

  it('deletes the warband after confirmation and returns to the list', async () => {
    const fixture = setup();
    http.expectOne('/api/warbands/wb-1').flush(makeWarband({ name: 'Da Boyz' }));
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    vi.spyOn(TestBed.inject(MatDialog), 'open').mockReturnValue({
      afterClosed: () => of(true),
    } as never);
    await fixture.whenStable();
    fixture.detectChanges();

    [...html(fixture).querySelectorAll('button')]
      .find((b) => b.textContent?.includes('Delete'))
      ?.click();
    await vi.waitFor(() =>
      http.expectOne('/api/warbands/wb-1').flush(null, { status: 204, statusText: 'No Content' }),
    );
    await vi.waitFor(() => expect(navigate).toHaveBeenCalledWith(['/warbands']));
    expect(notify.info).toHaveBeenCalledWith('Da Boyz was deleted.');
  });
});
