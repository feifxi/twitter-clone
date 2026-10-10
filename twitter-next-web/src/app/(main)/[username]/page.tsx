import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { dehydrate, HydrationBoundary, QueryClient } from '@tanstack/react-query';
import { getUserByUsername, prefetchFirstPage } from '@/api/server';
import { PAGE_SIZE, queryKeys } from '@/lib/query-keys';
import { Profile } from './Profile';

type Props = { params: Promise<{ username: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const username = decodeURIComponent((await params).username);
  const user = await getUserByUsername(username);
  if (!user) return { title: 'Profile not found' };

  const name = `${user.displayName ?? user.username} (@${user.username})`;
  return {
    title: `${name} / Chanom Twitter`,
    description: user.bio ?? undefined,
    openGraph: {
      title: name,
      description: user.bio ?? undefined,
      images: user.avatarUrl ? [user.avatarUrl] : undefined,
    },
  };
}

export default async function ProfilePage({ params }: Props) {
  const username = decodeURIComponent((await params).username);
  const user = await getUserByUsername(username);
  if (!user) notFound();

  const queryClient = new QueryClient();
  queryClient.setQueryData(queryKeys.users.byUsername(username), user);
  await prefetchFirstPage(queryClient, queryKeys.users.timeline(user.id, 'posts'), `/users/${user.id}/posts?size=${PAGE_SIZE}`);

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <Profile username={username} />
    </HydrationBoundary>
  );
}
