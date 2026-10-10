'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Search, X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SearchBoxProps {
  /** Starting text, e.g. the current ?q=. */
  initial?: string;
  autoFocus?: boolean;
  className?: string;
}

/** The pill search field; submitting opens /search?q=…. */
export function SearchBox({ initial = '', autoFocus, className }: SearchBoxProps) {
  const router = useRouter();
  const [value, setValue] = useState(initial);

  return (
    <form
      role="search"
      className={className}
      onSubmit={(e) => {
        e.preventDefault();
        const q = value.trim();
        if (q) router.push(`/search?q=${encodeURIComponent(q)}`);
      }}
    >
      <label className="group flex h-11 items-center gap-3 rounded-full border border-transparent bg-muted px-4 focus-within:border-primary focus-within:bg-background">
        <Search className="size-[18px] shrink-0 text-muted-foreground group-focus-within:text-primary" />
        <input
          type="text"
          inputMode="search"
          enterKeyHint="search"
          placeholder="Search"
          aria-label="Search"
          autoFocus={autoFocus}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          className="min-w-0 flex-1 bg-transparent text-[15px] outline-none placeholder:text-muted-foreground"
        />
        {value && (
          <button
            type="button"
            onClick={() => setValue('')}
            aria-label="Clear search"
            className={cn('flex size-5 cursor-pointer items-center justify-center rounded-full bg-primary text-primary-foreground')}
          >
            <X className="size-3.5" />
          </button>
        )}
      </label>
    </form>
  );
}
