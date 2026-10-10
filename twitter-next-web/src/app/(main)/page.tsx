import { dehydrate, HydrationBoundary, QueryClient } from '@tanstack/react-query';
import { prefetchFirstPage } from '@/api/server';
import { PAGE_SIZE, queryKeys } from '@/lib/query-keys';
import { Home } from './Home';

type Props = { searchParams: Promise<{ tab?: string }> };

export default async function HomePage({ searchParams }: Props) {
  const queryClient = new QueryClient();
  // The "For you" tab is the default; the Following tab loads in the browser.
  if ((await searchParams).tab !== 'following') {
    await prefetchFirstPage(queryClient, queryKeys.feeds.global, `/feeds/global?size=${PAGE_SIZE}`);
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <Home />
    </HydrationBoundary>
  );
}
