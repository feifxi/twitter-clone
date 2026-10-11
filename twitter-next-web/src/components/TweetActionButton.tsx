import { formatCount } from '@/lib/format';
import { cn } from '@/lib/utils';

// Hover and active colors per action, from the theme tokens.
const TONES = {
  primary: { active: 'text-primary', hover: 'group-hover:bg-primary/10 group-hover:text-primary' },
  retweet: { active: 'text-retweet', hover: 'group-hover:bg-retweet/10 group-hover:text-retweet' },
  like: { active: 'text-like', hover: 'group-hover:bg-like/10 group-hover:text-like' },
} as const;

interface TweetActionButtonProps {
  icon: React.ReactNode;
  label: string;
  count?: number;
  active?: boolean;
  tone?: keyof typeof TONES;
  onClick: (e: React.MouseEvent) => void;
}

/** One reply/retweet/like/share button: icon in a hover circle plus an optional count. */
export function TweetActionButton({ icon, label, count, active = false, tone = 'primary', onClick }: TweetActionButtonProps) {
  const colors = TONES[tone];
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      aria-pressed={tone === 'primary' ? undefined : active}
      className={cn(
        'group flex cursor-pointer items-center gap-0.5 text-muted-foreground transition-colors',
        active && colors.active,
      )}
    >
      <span className={cn('rounded-full p-2 transition-all duration-150 active:scale-90 [&_svg]:size-[18px]', colors.hover)}>
        {icon}
      </span>
      <span className={cn('min-w-[2ch] text-[13px] tabular-nums transition-colors', colors.hover.split(' ')[1])}>
        {count ? formatCount(count) : ''}
      </span>
    </button>
  );
}
