import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { makeWarband } from '../../../testing/factories';
import { submitForm, typeInto } from '../../../testing/dom';
import { API_URL } from '../../api/api-url';
import { Warband } from '../../api/models';
import { WarbandFormDialog } from './warband-form-dialog';

describe('WarbandFormDialog', () => {
  let http: HttpTestingController;
  const close = vi.fn();

  function setup(existing: Warband | null) {
    close.mockReset();
    TestBed.configureTestingModule({
      imports: [WarbandFormDialog],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: API_URL, useValue: '/api' },
        { provide: MatDialogRef, useValue: { close } },
        { provide: MAT_DIALOG_DATA, useValue: existing },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(WarbandFormDialog);
    fixture.detectChanges();
    return fixture;
  }

  afterEach(() => http.verify());

  it('creates a warband with the entered values', async () => {
    const fixture = setup(null);
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[formControlName=name]', 'Da Boyz');
    typeInto(html, 'input[formControlName=faction]', 'Orks');
    typeInto(html, 'input[formControlName=requisition_points]', '5');
    submitForm(html);

    const req = http.expectOne('/api/warbands/create');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({
      name: 'Da Boyz',
      faction: 'Orks',
      description: '',
      requisition_points: 5,
      supply_limit: 0,
    });
    const created = makeWarband({ name: 'Da Boyz' });
    req.flush(created, { status: 201, statusText: 'Created' });
    await vi.waitFor(() => expect(close).toHaveBeenCalledWith(created));
  });

  it('is prefilled when editing, and saves with PATCH', async () => {
    const fixture = setup(makeWarband({ id: 'wb-9', name: 'Old name', requisition_points: 7 }));
    const html = fixture.nativeElement as HTMLElement;
    expect(html.querySelector<HTMLInputElement>('input[formControlName=name]')?.value).toBe(
      'Old name',
    );
    expect(
      html.querySelector<HTMLInputElement>('input[formControlName=requisition_points]')?.value,
    ).toBe('7');

    typeInto(html, 'input[formControlName=name]', 'New name');
    submitForm(html);
    const req = http.expectOne('/api/warbands/wb-9');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toMatchObject({ name: 'New name', requisition_points: 7 });
    req.flush(makeWarband({ id: 'wb-9', name: 'New name' }));
    await vi.waitFor(() => expect(close).toHaveBeenCalled());
  });

  it.each([
    ['a blank name', 'name', '   ', 'Cannot be only spaces'],
    ['a name over 100 characters', 'name', 'n'.repeat(101), 'At most 100 characters'],
    ['a negative requisition', 'requisition_points', '-1', 'Must be at least 0'],
    ['requisition over a million', 'requisition_points', '1000001', 'Must be at most 1000000'],
    ['a decimal supply limit', 'supply_limit', '2.5', 'Whole numbers only'],
  ])('rejects %s without calling the API', async (_label, field, value, message) => {
    const fixture = setup(null);
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[formControlName=name]', 'Da Boyz');
    typeInto(html, `input[formControlName=${field}]`, value);
    submitForm(html);
    await fixture.whenStable();
    fixture.detectChanges();

    http.expectNone('/api/warbands/create');
    expect(close).not.toHaveBeenCalled();
    expect(html.textContent).toContain(message);
  });

  it('shows the API reason when the warband limit is reached', async () => {
    const fixture = setup(null);
    const html = fixture.nativeElement as HTMLElement;
    typeInto(html, 'input[formControlName=name]', 'One too many');
    submitForm(html);
    http
      .expectOne('/api/warbands/create')
      .flush(
        { error: 'limit reached: at most 50 warbands per user' },
        { status: 409, statusText: 'Conflict' },
      );

    await vi.waitFor(() => {
      fixture.detectChanges();
      expect(html.querySelector('[role=alert]')?.textContent).toContain(
        'Limit reached: at most 50 warbands per user',
      );
    });
    expect(close).not.toHaveBeenCalled();
  });
});
