import type { Metadata } from 'next';
import { HydrationBoundary } from '@tanstack/react-query';
import { prefetchProfileList, requireProfileUser } from '../data';
import { FollowList } from '../FollowList';

type Props = { params: Promise<{ username: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { user } = await requireProfileUser(params);
  return { title: `People following ${user.displayName || user.username} (@${user.username}) / Chanom Twitter` };
}

export default async function FollowersPage({ params }: Props) {
  const { username, state } = await prefetchProfileList(params, 'followers');
  return (
    <HydrationBoundary state={state}>
      <FollowList username={username} kind="followers" />
    </HydrationBoundary>
  );
}
