import Link from 'next/link';
import { cn } from '@/lib/utils';

export interface Tab {
  label: string;
  active: boolean;
  /** A route (tabs as URLs, so they can be shared and survive reloads)… */
  href?: string;
  /** …or a click handler. */
  onSelect?: () => void;
  disabled?: boolean;
}

/** X-style tab row: equal-width tabs with a blue pill under the active label. */
/** `replace` swaps the history entry instead of pushing one (e.g. feed filters). */
export function TabBar({ tabs, replace = false, className }: { tabs: Tab[]; replace?: boolean; className?: string }) {
  return (
    <nav className={cn('flex', className)} role="tablist">
      {tabs.map((tab) => {
        const content = (
          <span className="relative inline-flex h-full items-center">
            {tab.label}
            {tab.active && <span className="absolute inset-x-0 bottom-0 h-1 rounded-full bg-primary" />}
          </span>
        );
        const className = cn(
          'flex h-[53px] flex-1 justify-center px-4 text-[15px] transition-colors hover:bg-accent',
          tab.active ? 'font-bold text-foreground' : 'font-medium text-muted-foreground',
          tab.disabled && 'pointer-events-none opacity-50',
        );
        return tab.href ? (
          <Link key={tab.label} href={tab.href} role="tab" aria-selected={tab.active} replace={replace} scroll={false} className={className}>
            {content}
          </Link>
        ) : (
          <button
            key={tab.label}
            type="button"
            role="tab"
            aria-selected={tab.active}
            disabled={tab.disabled}
            onClick={tab.onSelect}
            className={cn(className, 'cursor-pointer')}
          >
            {content}
          </button>
        );
      })}
    </nav>
  );
}
