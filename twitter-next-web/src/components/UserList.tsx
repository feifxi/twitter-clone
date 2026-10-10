'use client';

import type { InfiniteData, UseInfiniteQueryResult } from '@tanstack/react-query';
import { EmptyState, ErrorState, type EmptyCopy } from '@/components/ListStates';
import { UserRow, UserRowSkeleton } from '@/components/UserRow';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import type { PageResponse, UserResponse } from '@/types';

interface UserListProps {
  query: UseInfiniteQueryResult<InfiniteData<PageResponse<UserResponse>>>;
  empty: EmptyCopy;
}

/** An infinitely scrolling list of users with loading, error and empty states. */
export function UserList({ query, empty }: UserListProps) {
  const sentinel = useInfiniteScroll(query);
  const users = query.data?.pages.flatMap((page) => page.items) ?? [];

  if (query.isPending) return Array.from({ length: 6 }, (_, i) => <UserRowSkeleton key={i} />);
  if (query.isError && users.length === 0) return <ErrorState onRetry={() => query.refetch()} />;
  if (users.length === 0) return <EmptyState {...empty} />;

  return (
    <div>
      {users.map((user) => (
        <UserRow key={user.id} user={user} />
      ))}
      <div ref={sentinel} />
      {query.isFetchingNextPage && <UserRowSkeleton />}
    </div>
  );
}
