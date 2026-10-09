import { AbstractControl } from '@angular/forms';

/** The message to show under a form field, or an empty string when it is valid. */
export function errorMessage(control: AbstractControl | null): string {
  const errors = control?.errors;
  if (!errors) {
    return '';
  }
  if (errors['required']) {
    return 'Required';
  }
  if (errors['notBlank']) {
    return 'Cannot be only spaces';
  }
  if (errors['email']) {
    return 'Enter a valid email address';
  }
  if (errors['minlength']) {
    return `At least ${errors['minlength'].requiredLength} characters`;
  }
  if (errors['maxlength']) {
    return `At most ${errors['maxlength'].requiredLength} characters`;
  }
  if (errors['min'] || errors['max']) {
    const min = errors['min']?.min;
    const max = errors['max']?.max;
    return min !== undefined ? `Must be at least ${min}` : `Must be at most ${max}`;
  }
  if (errors['wholeNumber']) {
    return 'Whole numbers only';
  }
  if (errors['safeText']) {
    return 'Contains characters that are not allowed';
  }
  return 'Invalid value';
}
