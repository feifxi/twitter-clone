/*
 * Trailing debounce for toggle mutations (like, retweet, follow).
 *
 * The cache is updated on every click, so the button feels instant. The request
 * waits until the user stops clicking; then exactly one request is sent for the
 * final state, or none if they toggled back to where the server already is.
 * Spam-clicking like/unlike therefore never races N requests on the server.
 *
 * State is module-level, so a pending toggle survives the component unmounting
 * (click like, navigate away immediately: the request still goes out), and two
 * copies of the same tweet on screen share one timer.
 */

type Pending = {
  timer: ReturnType<typeof setTimeout>;
  /** What the server holds: captured on the first click of a cycle. */
  serverState: boolean;
  /** The latest state the user toggled to. */
  intendedState: boolean;
  sync: (state: boolean) => Promise<void>;
};

const pending = new Map<string, Pending>();
const DELAY_MS = 400;

function fire(scope: string) {
  const entry = pending.get(scope);
  if (!entry) return;
  pending.delete(scope);
  if (entry.intendedState === entry.serverState) return;
  // sync handles its own errors (rollback, toast).
  void entry.sync(entry.intendedState);
}

/**
 * Schedules (or reschedules) the request for a toggle.
 *
 * @param scope unique per toggleable thing, e.g. `like:42`
 * @param serverState the state before this click; used only on the first click of a cycle
 * @param intendedState the state the user just toggled to
 * @param sync sends the request for the final state
 */
export function debouncedToggle(
  scope: string,
  serverState: boolean,
  intendedState: boolean,
  sync: (state: boolean) => Promise<void>,
) {
  const existing = pending.get(scope);
  if (existing) clearTimeout(existing.timer);
  pending.set(scope, {
    timer: setTimeout(() => fire(scope), DELAY_MS),
    serverState: existing?.serverState ?? serverState,
    intendedState,
    sync,
  });
}
