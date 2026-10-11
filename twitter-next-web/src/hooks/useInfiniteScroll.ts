import { useEffect, useRef } from 'react';

type Pager = { hasNextPage: boolean; isFetchingNextPage: boolean; fetchNextPage: () => unknown };

/** Returns a ref for a sentinel element; the next page loads when it scrolls into view. */
export function useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage }: Pager) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el || !hasNextPage || isFetchingNextPage) return;
    const observer = new IntersectionObserver(([entry]) => entry?.isIntersecting && fetchNextPage(), { rootMargin: '400px' });
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);
  return ref;
}
