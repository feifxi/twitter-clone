/**
 * Every TanStack Query key in the app. Inline arrays scattered across hooks are
 * typo-prone (`['feed']` silently misses `['feeds']`), so hooks and cache
 * updates all build keys here. Keys are hierarchical: invalidating a prefix
 * such as `queryKeys.feeds.all` also hits every key below it.
 */

/** Page size of every list; the server prefetch and the client must agree on it. */
export const PAGE_SIZE = 20;

/** Profile timelines served by GET /users/:id/{posts,replies,media,likes}. */
export type ProfileTab = 'posts' | 'replies' | 'media' | 'likes';

export const queryKeys = {
  auth: {
    me: ['auth', 'me'] as const,
  },
  feeds: {
    all: ['feeds'] as const,
    global: ['feeds', 'global'] as const,
    following: ['feeds', 'following'] as const,
  },
  tweets: {
    all: ['tweets'] as const,
    detail: (id: number) => ['tweets', id] as const,
    replies: (id: number) => ['tweets', id, 'replies'] as const,
  },
  users: {
    all: ['users'] as const,
    detail: (id: number) => ['users', id] as const,
    byUsername: (username: string) => ['users', 'by-username', username] as const,
    timeline: (id: number, tab: ProfileTab) => ['users', id, 'timeline', tab] as const,
    followers: (id: number) => ['users', id, 'followers'] as const,
    following: (id: number) => ['users', id, 'following'] as const,
  },
  search: {
    all: ['search'] as const,
    tweets: (q: string) => ['search', 'tweets', q] as const,
    users: (q: string, size: number) => ['search', 'users', q, size] as const,
    people: (q: string) => ['search', 'people', q] as const,
    hashtags: (q: string) => ['search', 'hashtags', q] as const,
  },
  discovery: {
    all: ['discovery'] as const,
    trending: (limit: number) => ['discovery', 'trending', limit] as const,
    users: (size: number) => ['discovery', 'users', size] as const,
  },
  notifications: {
    all: ['notifications'] as const,
    list: ['notifications', 'list'] as const,
    unreadCount: ['notifications', 'unread-count'] as const,
  },
  messages: {
    all: ['messages'] as const,
    conversations: ['messages', 'conversations'] as const,
    conversation: (id: number) => ['messages', 'conversation', id] as const,
  },
  brainrot: ['brainrot'] as const,
} as const;
