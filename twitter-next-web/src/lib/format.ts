import { differenceInDays, differenceInHours, differenceInMinutes, differenceInSeconds, format } from 'date-fns';

/** 999 → "999", 1234 → "1.2K", 2_500_000 → "2.5M". */
export function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1).replace(/\.0$/, '')}K`;
  return String(n);
}

/** The name shown for a user: display name, else the handle. */
export function displayNameOf(user: { displayName?: string | null; username: string }): string {
  return user.displayName?.trim() || user.username;
}

/** Timeline timestamps: "45s", "12m", "3h", "5d", then "Mar 4" (or "Mar 4, 2025" in another year). */
export function formatRelativeTime(iso: string, now = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  const seconds = Math.max(0, differenceInSeconds(now, date));
  if (seconds < 60) return `${seconds}s`;
  const minutes = differenceInMinutes(now, date);
  if (minutes < 60) return `${minutes}m`;
  const hours = differenceInHours(now, date);
  if (hours < 24) return `${hours}h`;
  if (differenceInDays(now, date) < 7) return `${differenceInDays(now, date)}d`;
  return format(date, date.getFullYear() === now.getFullYear() ? 'MMM d' : 'MMM d, yyyy');
}

/** Tweet detail timestamp: "3:42 PM · Mar 4, 2026". */
export function formatFullTime(iso: string): string {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? '' : format(date, "h:mm a '·' MMM d, yyyy");
}

/**
 * A stable color per name for avatar fallbacks, as "rgb(...)" so the server and
 * the browser render the identical string (no hydration mismatch).
 */
export function avatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0;
  const h = Math.abs(hash) % 360;
  // HSL(h, 65%, 50%) → RGB
  const s = 0.65;
  const l = 0.5;
  const c = (1 - Math.abs(2 * l - 1)) * s;
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1));
  const m = l - c / 2;
  const [r, g, b] =
    h < 60 ? [c, x, 0] : h < 120 ? [x, c, 0] : h < 180 ? [0, c, x] : h < 240 ? [0, x, c] : h < 300 ? [x, 0, c] : [c, 0, x];
  const to255 = (v: number) => Math.round((v + m) * 255);
  return `rgb(${to255(r)}, ${to255(g)}, ${to255(b)})`;
}
