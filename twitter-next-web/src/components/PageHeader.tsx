'use client';

import { ArrowLeft } from 'lucide-react';
import { useRouter } from 'next/navigation';

export function BackButton() {
  const router = useRouter();
  return (
    <button
      type="button"
      onClick={() => (window.history.length > 1 ? router.back() : router.push('/'))}
      className="-ml-2 cursor-pointer rounded-full p-2 transition-colors hover:bg-accent"
      aria-label="Back"
    >
      <ArrowLeft className="size-5" />
    </button>
  );
}

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  back?: boolean;
  /** Extra content under the title row, e.g. a TabBar. */
  children?: React.ReactNode;
}

/** Sticky, blurred page header: optional back button, title and subtitle. */
export function PageHeader({ title, subtitle, back = false, children }: PageHeaderProps) {
  return (
    <header className="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur-md">
      <div className="flex min-h-[53px] items-center gap-6 px-4">
        {back && <BackButton />}
        <div className="min-w-0">
          <h1 className="truncate text-[20px] font-bold leading-6">{title}</h1>
          {subtitle && <p className="truncate text-[13px] text-muted-foreground">{subtitle}</p>}
        </div>
      </div>
      {children}
    </header>
  );
}
