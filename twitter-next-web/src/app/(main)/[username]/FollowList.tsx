'use client';

import { PageHeader } from '@/components/PageHeader';
import { TabBar } from '@/components/TabBar';
import { UserList } from '@/components/UserList';
import { useUserFollowers, useUserFollowing, useUserProfileByUsername } from '@/hooks/useProfile';
import { displayNameOf } from '@/lib/format';

export function FollowList({ username, kind }: { username: string; kind: 'followers' | 'following' }) {
  const { data: user } = useUserProfileByUsername(username);
  const userId = user?.id ?? null;
  // Hooks can't be called conditionally: both are declared, only the visible list's is enabled.
  const followers = useUserFollowers(kind === 'followers' ? userId : null);
  const following = useUserFollowing(kind === 'following' ? userId : null);

  const base = `/${username}`;
  return (
    <div className="min-h-screen">
      <PageHeader title={user ? displayNameOf(user) : username} subtitle={`@${username}`} back>
        <TabBar
          replace
          tabs={[
            { label: 'Followers', href: `${base}/followers`, active: kind === 'followers' },
            { label: 'Following', href: `${base}/following`, active: kind === 'following' },
          ]}
        />
      </PageHeader>
      {kind === 'followers' ? (
        <UserList query={followers} empty={{ title: 'Looking for followers?', body: `When someone follows @${username}, they'll be listed here.` }} />
      ) : (
        <UserList query={following} empty={{ title: `@${username} isn't following anyone`, body: 'Once they follow accounts, they’ll show up here.' }} />
      )}
    </div>
  );
}
