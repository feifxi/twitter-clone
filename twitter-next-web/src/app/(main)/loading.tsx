import { TweetSkeleton } from '@/components/Tweet';

/**
 * Shown inside the navigation while a page renders on the server (a reload, or a click to a page
 * that prefetches its data), so the click gets an answer before the data does.
 */
export default function Loading() {
  return (
    <div role="status" aria-label="Loading">
      <div className="h-[53px] border-b border-border" />
      {Array.from({ length: 6 }, (_, i) => (
        <TweetSkeleton key={i} />
      ))}
    </div>
  );
}
