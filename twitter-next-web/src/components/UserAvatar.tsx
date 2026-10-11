import Link from 'next/link';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { avatarColor, displayNameOf } from '@/lib/format';
import { cn } from '@/lib/utils';

const SIZES = {
  xs: { box: 'size-6', text: 'text-[10px]' },
  sm: { box: 'size-8', text: 'text-xs' },
  md: { box: 'size-10', text: 'text-sm' },
  lg: { box: 'size-16', text: 'text-xl' },
  xl: { box: 'size-[134px]', text: 'text-5xl' },
} as const;

interface UserAvatarProps {
  user: { username: string; displayName?: string | null; avatarUrl?: string | null };
  size?: keyof typeof SIZES;
  /** Links to the profile (and stops the click from reaching a clickable card). */
  link?: boolean;
  className?: string;
}

/** A user's picture, or their initials on a color derived from their name. */
export function UserAvatar({ user, size = 'md', link = false, className }: UserAvatarProps) {
  const name = displayNameOf(user);
  const initials = name
    .split(/\s+/)
    .map((word) => word[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();

  const avatar = (
    <Avatar className={cn(SIZES[size].box, className)}>
      {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={name} />}
      <AvatarFallback className={cn('font-semibold text-white', SIZES[size].text)} style={{ backgroundColor: avatarColor(user.username) }}>
        {initials || '?'}
      </AvatarFallback>
    </Avatar>
  );

  if (!link) return avatar;
  return (
    <Link
      href={`/${user.username}`}
      onClick={(e) => e.stopPropagation()}
      className="shrink-0 rounded-full transition-opacity hover:opacity-90"
      aria-label={`${name}'s profile`}
    >
      {avatar}
    </Link>
  );
}
