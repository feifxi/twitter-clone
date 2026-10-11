'use client';

import { Suspense } from 'react';
import { usePathname, useSearchParams } from 'next/navigation';
import { FeedList } from '@/components/FeedList';
import { EmptyState } from '@/components/ListStates';
import { BackButton } from '@/components/PageHeader';
import { SearchBox } from '@/components/SearchBox';
import { TabBar } from '@/components/TabBar';
import { UserList } from '@/components/UserList';
import { useSearchPeople, useSearchTweets } from '@/hooks/useDiscovery';

function SearchContent() {
  const searchParams = useSearchParams();
  const pathname = usePathname();
  const q = searchParams.get('q')?.trim() ?? '';
  const tab = searchParams.get('tab') === 'people' ? 'people' : 'top';

  const tweets = useSearchTweets(q, tab === 'top');
  const people = useSearchPeople(q, tab === 'people');

  const urlFor = (next: { q?: string; tab?: string }) => {
    const params = new URLSearchParams(searchParams);
    if (next.q !== undefined) params.set('q', next.q);
    if (next.tab) params.set('tab', next.tab);
    if (next.tab === 'top') params.delete('tab');
    return `${pathname}?${params}`;
  };

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur-md">
        <div className="flex items-center gap-4 px-4 py-1.5">
          <BackButton />
          <SearchBox initial={q} className="flex-1" />
        </div>
        {q && (
          <TabBar
            replace
            tabs={[
              { label: 'Top', href: urlFor({ tab: 'top' }), active: tab === 'top' },
              { label: 'People', href: urlFor({ tab: 'people' }), active: tab === 'people' },
            ]}
          />
        )}
      </header>

      {!q ? (
        <EmptyState title="Search Chanom Twitter" body="Find posts, hashtags and people." />
      ) : tab === 'top' ? (
        <FeedList query={tweets} empty={{ title: `No results for "${q}"`, body: 'Try searching for something else.' }} />
      ) : (
        <UserList query={people} empty={{ title: `No people found for "${q}"`, body: 'Try a name or @handle.' }} />
      )}
    </div>
  );
}

// Keyed on q so the search box resets to the URL's query after navigation.
function SearchPageInner() {
  const q = useSearchParams().get('q') ?? '';
  return <SearchContent key={q} />;
}

export default function SearchPage() {
  return (
    <Suspense>
      <SearchPageInner />
    </Suspense>
  );
}
