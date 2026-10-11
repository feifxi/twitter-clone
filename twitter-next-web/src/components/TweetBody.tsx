import Link from 'next/link';
import { Fragment } from 'react';

// #hashtag, @mention, or an http(s) URL.
const TOKEN = /(#\w+|@\w{1,20}|https?:\/\/[^\s]+)/g;

const stop = (e: React.MouseEvent) => e.stopPropagation();

/** Tweet text with hashtags, mentions and URLs as links (clicks don't open the tweet). */
export function TweetBody({ content, className }: { content: string; className?: string }) {
  return (
    <p className={className}>
      {content.split(TOKEN).map((part, i) => {
        if (/^#\w+$/.test(part)) {
          return (
            <Link key={i} href={`/search?q=${encodeURIComponent(part)}`} onClick={stop} className="text-primary hover:underline">
              {part}
            </Link>
          );
        }
        if (/^@\w{1,20}$/.test(part)) {
          return (
            <Link key={i} href={`/${part.slice(1)}`} onClick={stop} className="text-primary hover:underline">
              {part}
            </Link>
          );
        }
        if (/^https?:\/\//.test(part)) {
          return (
            <a key={i} href={part} target="_blank" rel="noopener noreferrer nofollow" onClick={stop} className="text-primary hover:underline">
              {part.replace(/^https?:\/\/(www\.)?/, '')}
            </a>
          );
        }
        return <Fragment key={i}>{part}</Fragment>;
      })}
    </p>
  );
}
