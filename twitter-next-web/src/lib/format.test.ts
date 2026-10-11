import { describe, expect, it } from 'vitest';
import { avatarColor, displayNameOf, formatCount, formatRelativeTime } from './format';

describe('formatCount', () => {
  it.each([
    [0, '0'],
    [999, '999'],
    [1000, '1K'],
    [1234, '1.2K'],
    [2_500_000, '2.5M'],
  ])('%d → %s', (n, want) => expect(formatCount(n)).toBe(want));
});

describe('formatRelativeTime', () => {
  const now = new Date('2026-06-15T12:00:00Z');
  it.each([
    ['2026-06-15T11:59:30Z', '30s'],
    ['2026-06-15T11:15:00Z', '45m'],
    ['2026-06-15T09:00:00Z', '3h'],
    ['2026-06-12T12:00:00Z', '3d'],
    ['2026-03-04T12:00:00Z', 'Mar 4'],
    ['2025-03-04T12:00:00Z', 'Mar 4, 2025'],
    ['2026-06-15T12:00:05Z', '0s'], // clock skew: never negative
    ['not a date', ''],
  ])('%s → %s', (iso, want) => expect(formatRelativeTime(iso, now)).toBe(want));
});

describe('displayNameOf', () => {
  it('falls back to the handle for empty names', () => {
    expect(displayNameOf({ displayName: '  ', username: 'jo' })).toBe('jo');
    expect(displayNameOf({ displayName: 'Jo', username: 'jo' })).toBe('Jo');
  });
});

describe('avatarColor', () => {
  it('is stable per name and a valid rgb()', () => {
    expect(avatarColor('alice')).toBe(avatarColor('alice'));
    expect(avatarColor('alice')).toMatch(/^rgb\(\d{1,3}, \d{1,3}, \d{1,3}\)$/);
  });
});
