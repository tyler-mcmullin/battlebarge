import { safeReturnUrl } from './return-url';

describe('safeReturnUrl', () => {
  it('allows paths inside the app, including query strings', () => {
    expect(safeReturnUrl('/warbands/abc')).toBe('/warbands/abc');
    expect(safeReturnUrl('/warbands?x=1')).toBe('/warbands?x=1');
  });

  it.each([
    null,
    '',
    'warbands',
    'https://evil.example',
    'http://evil.example/x',
    '//evil.example',
    '/\\evil.example',
    'javascript:alert(1)',
  ])('falls back for %s', (url) => {
    expect(safeReturnUrl(url)).toBe('/warbands');
  });

  it('uses the fallback it is given', () => {
    expect(safeReturnUrl('//evil.example', '/home')).toBe('/home');
  });
});
