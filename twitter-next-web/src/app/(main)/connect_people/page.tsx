'use client';

import { EmptyState, ErrorState } from '@/components/ListStates';
import { PageHeader } from '@/components/PageHeader';
import { UserRow, UserRowSkeleton } from '@/components/UserRow';
import { useSuggestedUsers } from '@/hooks/useDiscovery';

export default function ConnectPeoplePage() {
  const suggestions = useSuggestedUsers(20);
  const users = suggestions.data?.items ?? [];

  return (
    <div className="min-h-screen">
      <PageHeader title="Connect" back />
      <h2 className="px-4 py-3 text-[20px] font-extrabold">Suggested for you</h2>
      {suggestions.isPending ? (
        Array.from({ length: 6 }, (_, i) => <UserRowSkeleton key={i} />)
      ) : suggestions.isError ? (
        <ErrorState onRetry={() => suggestions.refetch()} />
      ) : users.length === 0 ? (
        <EmptyState title="You're all caught up" body="New suggestions show up here as more people join." />
      ) : (
        users.map((user) => <UserRow key={user.id} user={user} />)
      )}
    </div>
  );
}
