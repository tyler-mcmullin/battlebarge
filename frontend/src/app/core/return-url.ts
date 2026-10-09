/**
 * Where to go after signing in. Only paths inside this app are allowed, so a
 * crafted link cannot send someone to another site after they log in.
 */
export function safeReturnUrl(url: string | null, fallback = '/warbands'): string {
  if (!url || !url.startsWith('/') || url.startsWith('//') || url.includes('\\')) {
    return fallback;
  }
  return url;
}
