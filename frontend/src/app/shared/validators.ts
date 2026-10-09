import { AbstractControl, ValidationErrors } from '@angular/forms';

/** Rejects text that is only spaces (the API's `notblank` rule). Pair it with Validators.required. */
export function notBlank(control: AbstractControl): ValidationErrors | null {
  const value = control.value;
  return typeof value === 'string' && value.length > 0 && value.trim() === ''
    ? { notBlank: true }
    : null;
}

// Control characters other than tab, newline and carriage return (the API's `safetext` rule).
const UNSAFE_TEXT = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/;

/** Rejects NUL and other control characters, which the API refuses. */
export function safeText(control: AbstractControl): ValidationErrors | null {
  const value = control.value;
  return typeof value === 'string' && UNSAFE_TEXT.test(value) ? { safeText: true } : null;
}

/** Whole numbers only (no decimals). Empty values are left to Validators.required. */
export function wholeNumber(control: AbstractControl): ValidationErrors | null {
  const value = control.value;
  if (value === null || value === undefined || value === '') {
    return null;
  }
  return Number.isInteger(Number(value)) ? null : { wholeNumber: true };
}
