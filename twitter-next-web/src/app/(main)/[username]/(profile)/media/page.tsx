import { HydrationBoundary } from '@tanstack/react-query';
import { prefetchProfileList } from '../../data';
import { ProfileTimeline } from '../../ProfileTimeline';

export default async function ProfileMediaPage({ params }: { params: Promise<{ username: string }> }) {
  const { username, state } = await prefetchProfileList(params, 'media');
  return (
    <HydrationBoundary state={state}>
      <ProfileTimeline username={username} tab="media" />
    </HydrationBoundary>
  );
}
