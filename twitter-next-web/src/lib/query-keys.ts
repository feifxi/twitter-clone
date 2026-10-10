/**
 * Every TanStack Query key in the app. Keys are hierarchical, so invalidating a
 * prefix (e.g. `queryKeys.tweets.all`) also hits everything below it.
 */
export const queryKeys = {
  auth: {
    me: ['auth', 'me'] as const,
  },
} as const;
