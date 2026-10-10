import { useQueryClient, type InfiniteData } from '@tanstack/react-query';
import type { TweetResponse, PageResponse } from '@/types';

type TweetUpdater = (t: TweetResponse) => TweetResponse | null;

/**
 * Applies an update to every cached copy of a tweet. A tweet can appear in many
 * lists (and twice in one timeline when someone retweeted it), so each matching
 * item is updated; returning null from the updater removes that item.
 */
export function useUpdateTweetCache() {
    const queryClient = useQueryClient();

    return (tweetId: number, updater: TweetUpdater) => {
        const updateList = (items: TweetResponse[]) =>
            items
                .map((t) => (t.id === tweetId ? updater(t) : t))
                .filter((t): t is TweetResponse => t !== null);

        const updatePages = (old: InfiniteData<PageResponse<TweetResponse>> | undefined) =>
            old && { ...old, pages: old.pages.map((page) => ({ ...page, items: updateList(page.items) })) };

        for (const prefix of [['feeds'], ['search', 'tweets'], ['users']]) {
            queryClient.setQueriesData<InfiniteData<PageResponse<TweetResponse>>>(
                { queryKey: prefix, predicate: (q) => Array.isArray((q.state.data as { pages?: unknown })?.pages) },
                updatePages,
            );
        }

        // Tweet detail ['tweets', id] and reply lists ['tweets', id, 'replies'].
        queryClient.setQueriesData<unknown>({ queryKey: ['tweets'] }, (old: unknown) => {
            if (!old) return old;
            if ((old as InfiniteData<PageResponse<TweetResponse>>).pages) {
                return updatePages(old as InfiniteData<PageResponse<TweetResponse>>);
            }
            const t = old as TweetResponse;
            return t.id === tweetId ? (updater(t) ?? undefined) : old;
        });
    };
}
