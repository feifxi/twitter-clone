'use client';

import { Suspense, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Search } from 'lucide-react';
import { FeedList } from '@/components/FeedList';
import { EmptyState } from '@/components/ListStates';
import { BackButton } from '@/components/PageHeader';
import { TabBar } from '@/components/TabBar';
import { UserList } from '@/components/UserList';
import { useSearchPeople, useSearchTweets } from '@/hooks/useDiscovery';

function SearchContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const q = searchParams.get('q')?.trim() ?? '';
  const tab = searchParams.get('tab') === 'people' ? 'people' : 'top';
  // Re-keyed on q (below), so the box resets when the URL changes.
  const [input, setInput] = useState(q);

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
          <form
            className="flex-1"
            role="search"
            onSubmit={(e) => {
              e.preventDefault();
              if (input.trim()) router.push(urlFor({ q: input.trim() }));
            }}
          >
            <label className="flex items-center gap-3 rounded-full border border-transparent bg-muted px-4 py-2.5 focus-within:border-primary focus-within:bg-background">
              <Search className="size-[18px] shrink-0 text-muted-foreground" />
              <input
                type="search"
                placeholder="Search"
                aria-label="Search query"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                className="min-w-0 flex-1 bg-transparent text-[15px] outline-none placeholder:text-muted-foreground"
              />
            </label>
          </form>
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
