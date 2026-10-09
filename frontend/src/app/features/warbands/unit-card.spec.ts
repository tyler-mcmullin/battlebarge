import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material/dialog';
import { of } from 'rxjs';
import { makePerk, makeUnit } from '../../../testing/factories';
import { API_URL } from '../../api/api-url';
import { Unit } from '../../api/models';
import { Notify } from '../../core/notify';
import { UnitCard } from './unit-card';

describe('UnitCard', () => {
  let http: HttpTestingController;
  const notify = { info: vi.fn(), error: vi.fn() };
  const changed = vi.fn();
  const deleted = vi.fn();

  function setup(unit: Unit) {
    notify.error.mockReset();
    changed.mockReset();
    deleted.mockReset();
    TestBed.configureTestingModule({
      imports: [UnitCard],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: Notify, useValue: notify },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(UnitCard);
    fixture.componentRef.setInput('unit', unit);
    fixture.componentInstance.changed.subscribe(changed);
    fixture.componentInstance.deleted.subscribe(deleted);
    fixture.detectChanges();
    return fixture;
  }

  const html = (f: { nativeElement: unknown }) => f.nativeElement as HTMLElement;
  const text = (f: { nativeElement: unknown }, testId: string) =>
    html(f).querySelector(`[data-testid=${testId}]`)?.textContent?.trim();
  const click = (f: { nativeElement: unknown }, label: string) =>
    html(f).querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)?.click();

  afterEach(() => http.verify());

  it('shows the unit', () => {
    const fixture = setup(
      makeUnit({ unit_name: 'Boss', narrative_name: 'Grukk', points: 80, kills: 2, experience: 5 }),
    );
    expect(html(fixture).textContent).toContain('Boss');
    expect(html(fixture).textContent).toContain('Grukk');
    expect(html(fixture).textContent).toContain('80 pts');
    expect(text(fixture, 'kills')).toBe('2');
    expect(text(fixture, 'xp')).toBe('5');
  });

  it('adds a kill: sends +1 and shows the number the API returns', async () => {
    const fixture = setup(makeUnit({ kills: 2 }));
    click(fixture, 'Add a kill');

    const req = http.expectOne('/api/units/unit-1/kills');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ amount: 1 });
    req.flush(makeUnit({ kills: 3 }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(text(fixture, 'kills')).toBe('3');
    expect(changed).toHaveBeenCalledTimes(1);
  });

  it('removes experience with -1', async () => {
    const fixture = setup(makeUnit({ experience: 5 }));
    click(fixture, 'Remove an experience point');
    const req = http.expectOne('/api/units/unit-1/xp');
    expect(req.request.body).toEqual({ amount: -1 });
    req.flush(makeUnit({ experience: 4 }));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(text(fixture, 'xp')).toBe('4');
  });

  it('cannot go below zero: the minus buttons are disabled at 0', () => {
    const fixture = setup(makeUnit({ kills: 0, experience: 0 }));
    expect(
      html(fixture).querySelector<HTMLButtonElement>('button[aria-label="Remove a kill"]')
        ?.disabled,
    ).toBe(true);
    expect(
      html(fixture).querySelector<HTMLButtonElement>(
        'button[aria-label="Remove an experience point"]',
      )?.disabled,
    ).toBe(true);
    expect(
      html(fixture).querySelector<HTMLButtonElement>('button[aria-label="Add a kill"]')?.disabled,
    ).toBe(false);
  });

  it('keeps the old number and tells the user when the API refuses', async () => {
    const fixture = setup(makeUnit({ kills: 2 }));
    click(fixture, 'Add a kill');
    http
      .expectOne('/api/units/unit-1/kills')
      .flush({ error: 'invalid request body' }, { status: 400, statusText: 'Bad Request' });
    await vi.waitFor(() => expect(notify.error).toHaveBeenCalledWith('Invalid request body'));
    fixture.detectChanges();
    expect(text(fixture, 'kills')).toBe('2');
    expect(changed).not.toHaveBeenCalled();
  });

  it('marks scars differently from perks, and removes a perk by id', async () => {
    const perk = makePerk({ id: 'p1', name: 'Tough' });
    const scar = makePerk({ id: 'p2', name: 'Lost an eye', is_scar: true });
    const fixture = setup(makeUnit({ perks: [perk, scar] }));
    const items = html(fixture).querySelectorAll('.perk');
    expect(items.length).toBe(2);
    expect(items[0].classList.contains('scar')).toBe(false);
    expect(items[1].classList.contains('scar')).toBe(true);
    expect(items[1].textContent).toContain('Scar: Lost an eye');

    click(fixture, 'Remove Tough');
    const req = http.expectOne('/api/units/unit-1/perk/p1');
    expect(req.request.method).toBe('DELETE');
    req.flush(makeUnit({ perks: [scar] }));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(html(fixture).querySelectorAll('.perk').length).toBe(1);
  });

  it('deletes the unit after confirmation, and not without it', async () => {
    const fixture = setup(makeUnit());
    const open = vi.spyOn(TestBed.inject(MatDialog), 'open');

    open.mockReturnValueOnce({ afterClosed: () => of(false) } as never);
    await (fixture.componentInstance as unknown as { remove(): Promise<void> }).remove();
    http.expectNone('/api/units/unit-1');
    expect(deleted).not.toHaveBeenCalled();

    open.mockReturnValueOnce({ afterClosed: () => of(true) } as never);
    const done = (fixture.componentInstance as unknown as { remove(): Promise<void> }).remove();
    await vi.waitFor(() =>
      http.expectOne('/api/units/unit-1').flush(null, { status: 204, statusText: 'No Content' }),
    );
    await done;
    expect(deleted).toHaveBeenCalledTimes(1);
  });
});
