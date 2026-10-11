'use client';

import Link from 'next/link';
import { EmptyState, ErrorState } from '@/components/ListStates';
import { SearchBox } from '@/components/SearchBox';
import { useTrendingHashtags } from '@/hooks/useDiscovery';
import { formatCount } from '@/lib/format';

export default function ExplorePage() {
  const trending = useTrendingHashtags(20);

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-30 border-b border-border bg-background/80 px-4 py-1.5 backdrop-blur-md">
        <SearchBox />
      </header>

      <h1 className="px-4 py-3 text-[20px] font-extrabold">Trends for you</h1>
      {trending.isPending ? (
        Array.from({ length: 6 }, (_, i) => (
          <div key={i} className="animate-pulse space-y-2 px-4 py-3" aria-hidden="true">
            <div className="h-3 w-24 rounded bg-muted" />
            <div className="h-4 w-40 rounded bg-muted" />
          </div>
        ))
      ) : trending.isError ? (
        <ErrorState onRetry={() => trending.refetch()} />
      ) : !trending.data?.length ? (
        <EmptyState title="No trends yet" body="Hashtags people use in their posts show up here." />
      ) : (
        <ol>
          {trending.data.map((tag, i) => (
            <li key={tag.text}>
              <Link href={`/search?q=${encodeURIComponent(`#${tag.text}`)}`} className="block px-4 py-3 transition-colors hover:bg-foreground/[0.03]">
                <p className="text-[13px] text-muted-foreground">{i + 1} · Trending</p>
                <p className="text-[15px] font-bold">#{tag.text}</p>
                <p className="text-[13px] text-muted-foreground">
                  {formatCount(tag.usageCount)} {tag.usageCount === 1 ? 'post' : 'posts'}
                </p>
              </Link>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
