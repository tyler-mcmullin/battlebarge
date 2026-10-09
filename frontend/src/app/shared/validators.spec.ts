import { FormControl } from '@angular/forms';
import { errorMessage } from './form-errors';
import { notBlank, safeText, wholeNumber } from './validators';

describe('notBlank', () => {
  it('rejects text that is only spaces', () => {
    expect(notBlank(new FormControl('   '))).toEqual({ notBlank: true });
    expect(notBlank(new FormControl('\t \n'))).toEqual({ notBlank: true });
  });

  it('accepts real text, and leaves empty to Validators.required', () => {
    expect(notBlank(new FormControl(' a '))).toBeNull();
    expect(notBlank(new FormControl(''))).toBeNull();
  });
});

describe('safeText', () => {
  it('rejects NUL and other control characters', () => {
    expect(safeText(new FormControl('a\u0000b'))).toEqual({ safeText: true });
    expect(safeText(new FormControl('a\u0007b'))).toEqual({ safeText: true });
    expect(safeText(new FormControl('a\u007fb'))).toEqual({ safeText: true });
  });

  it('accepts tabs, newlines, and non-English text', () => {
    expect(safeText(new FormControl('line one\n\tline two'))).toBeNull();
    expect(safeText(new FormControl('héros 戦士'))).toBeNull();
  });
});

describe('wholeNumber', () => {
  it('rejects decimals', () => {
    expect(wholeNumber(new FormControl(1.5))).toEqual({ wholeNumber: true });
  });

  it('accepts integers (including 0 and negatives) and empty values', () => {
    expect(wholeNumber(new FormControl(0))).toBeNull();
    expect(wholeNumber(new FormControl(-3))).toBeNull();
    expect(wholeNumber(new FormControl(null))).toBeNull();
    expect(wholeNumber(new FormControl(''))).toBeNull();
  });
});

describe('errorMessage', () => {
  it('is empty for a valid control', () => {
    expect(errorMessage(new FormControl('ok'))).toBe('');
    expect(errorMessage(null)).toBe('');
  });

  it('describes each kind of problem', () => {
    const control = (errors: Record<string, unknown>) => {
      const c = new FormControl('x');
      c.setErrors(errors);
      return c;
    };
    expect(errorMessage(control({ required: true }))).toBe('Required');
    expect(errorMessage(control({ notBlank: true }))).toBe('Cannot be only spaces');
    expect(errorMessage(control({ email: true }))).toBe('Enter a valid email address');
    expect(errorMessage(control({ minlength: { requiredLength: 8 } }))).toBe(
      'At least 8 characters',
    );
    expect(errorMessage(control({ maxlength: { requiredLength: 100 } }))).toBe(
      'At most 100 characters',
    );
    expect(errorMessage(control({ min: { min: 0 } }))).toBe('Must be at least 0');
    expect(errorMessage(control({ max: { max: 5 } }))).toBe('Must be at most 5');
    expect(errorMessage(control({ wholeNumber: true }))).toBe('Whole numbers only');
    expect(errorMessage(control({ safeText: true }))).toBe(
      'Contains characters that are not allowed',
    );
    expect(errorMessage(control({ other: true }))).toBe('Invalid value');
  });
});
