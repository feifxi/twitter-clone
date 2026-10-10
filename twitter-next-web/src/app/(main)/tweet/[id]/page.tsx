import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { dehydrate, HydrationBoundary, QueryClient } from '@tanstack/react-query';
import { getTweet, prefetchFirstPage } from '@/api/server';
import { PAGE_SIZE, queryKeys } from '@/lib/query-keys';
import { TweetDetail } from './TweetDetail';

type Props = { params: Promise<{ id: string }> };

function parseId(id: string): number | null {
  const n = Number(id);
  return Number.isSafeInteger(n) && n > 0 ? n : null;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const id = parseId((await params).id);
  const tweet = id ? await getTweet(id) : null;
  if (!tweet) return { title: 'Post not found' };

  const author = tweet.user.displayName ?? tweet.user.username;
  const text = tweet.content?.slice(0, 140) ?? '';
  return {
    title: `${author} on Chanom Twitter: "${text}"`,
    description: text,
    openGraph: {
      title: `${author} (@${tweet.user.username})`,
      description: text,
      images: tweet.mediaType === 'IMAGE' && tweet.mediaUrl ? [tweet.mediaUrl] : undefined,
    },
  };
}

export default async function TweetPage({ params }: Props) {
  const id = parseId((await params).id);
  const tweet = id ? await getTweet(id) : null;
  if (!id || !tweet) notFound();

  const queryClient = new QueryClient();
  queryClient.setQueryData(queryKeys.tweets.detail(id), tweet);
  await prefetchFirstPage(queryClient, queryKeys.tweets.replies(id), `/tweets/${id}/replies?size=${PAGE_SIZE}`);

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <TweetDetail tweetId={id} />
    </HydrationBoundary>
  );
}
