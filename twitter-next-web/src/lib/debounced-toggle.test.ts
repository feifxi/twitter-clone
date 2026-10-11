import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { debouncedToggle } from './debounced-toggle';

describe('debouncedToggle', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('sends one request with the final state after the clicks stop', () => {
    const sync = vi.fn(async () => {});
    debouncedToggle('like:1', false, true, sync);
    vi.advanceTimersByTime(200);
    debouncedToggle('like:1', true, false, sync);
    vi.advanceTimersByTime(200);
    debouncedToggle('like:1', false, true, sync);
    expect(sync).not.toHaveBeenCalled();

    vi.advanceTimersByTime(400);
    expect(sync).toHaveBeenCalledTimes(1);
    expect(sync).toHaveBeenCalledWith(true);
  });

  it('sends nothing when the user toggles back to the server state', () => {
    const sync = vi.fn(async () => {});
    debouncedToggle('like:2', false, true, sync);
    debouncedToggle('like:2', true, false, sync); // serverState of later clicks is ignored
    vi.advanceTimersByTime(1_000);
    expect(sync).not.toHaveBeenCalled();
  });

  it('keeps separate timers per scope', () => {
    const sync = vi.fn(async () => {});
    debouncedToggle('like:3', false, true, sync);
    debouncedToggle('retweet:3', false, true, sync);
    vi.advanceTimersByTime(400);
    expect(sync).toHaveBeenCalledTimes(2);
  });

  it('uses the latest sync function', () => {
    const first = vi.fn(async () => {});
    const latest = vi.fn(async () => {});
    debouncedToggle('follow:4', false, true, first);
    debouncedToggle('follow:4', true, true, latest);
    vi.advanceTimersByTime(400);
    expect(first).not.toHaveBeenCalled();
    expect(latest).toHaveBeenCalledWith(true);
  });
});
