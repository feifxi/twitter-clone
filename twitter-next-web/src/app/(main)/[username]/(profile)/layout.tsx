import type { Metadata } from 'next';
import { dehydrate, HydrationBoundary, QueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-keys';
import { requireProfileUser } from '../data';
import { ProfileHeader } from '../ProfileHeader';

type Props = { params: Promise<{ username: string }>; children: React.ReactNode };

export async function generateMetadata({ params }: Omit<Props, 'children'>): Promise<Metadata> {
  const { user } = await requireProfileUser(params);
  const name = `${user.displayName || user.username} (@${user.username})`;
  return {
    title: `${name} / Chanom Twitter`,
    description: user.bio ?? undefined,
    openGraph: { title: name, description: user.bio ?? undefined, images: user.avatarUrl ? [user.avatarUrl] : undefined },
  };
}

/** Header and tabs stay mounted while switching tabs; each tab is its own route. */
export default async function ProfileLayout({ params, children }: Props) {
  const { username, user } = await requireProfileUser(params);
  const queryClient = new QueryClient();
  queryClient.setQueryData(queryKeys.users.byUsername(username), user);

  return (
    <div className="min-h-screen">
      <HydrationBoundary state={dehydrate(queryClient)}>
        <ProfileHeader username={username} />
      </HydrationBoundary>
      {children}
    </div>
  );
}
