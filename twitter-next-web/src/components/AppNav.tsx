'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Bell, BotMessageSquare, Home, LogIn, Mail, MoreHorizontal, Search, TvMinimalPlay, User, Users, type LucideIcon } from 'lucide-react';
import { ComposeTweetModal } from '@/components/ComposeTweetModal';
import { ConfirmModal } from '@/components/ConfirmModal';
import { ThemeToggle } from '@/components/ThemeToggle';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { UserAvatar } from '@/components/UserAvatar';
import { XLogo } from '@/components/XLogo';
import { useAuth } from '@/hooks/useAuth';
import { displayNameOf } from '@/lib/format';
import { cn } from '@/lib/utils';
import { useUIStore } from '@/store/useUIStore';

// Active icons are drawn filled, which only reads well for solid shapes; the rest go bold.
export const FILL_WHEN_ACTIVE = new Set<LucideIcon>([Home, Bell, User]);

/** "20+" past twenty, like X. */
export function badgeText(count: number) {
  return count > 20 ? '20+' : String(count);
}

/** Icon + label (label only on wide screens) + optional unread badge: the inside of every nav entry. */
function NavItemContent({ icon: Icon, label, active, badge }: { icon: LucideIcon; label: string; active: boolean; badge?: number }) {
  return (
    <span className="flex items-center gap-5 rounded-full p-3 transition-colors group-hover:bg-accent xl:pr-6">
      <span className="relative">
        <Icon className={cn('size-7', active && FILL_WHEN_ACTIVE.has(Icon) && 'fill-current')} strokeWidth={active ? 2.5 : 2} />
        {!!badge && (
          <span className="absolute -right-1.5 -top-1.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full border border-background bg-primary px-1 text-[11px] font-bold text-primary-foreground">
            {badgeText(badge)}
          </span>
        )}
      </span>
      <span className={cn('hidden text-[20px] leading-6 xl:inline', active && 'font-bold')}>{label}</span>
    </span>
  );
}

const itemClass = 'group flex w-fit cursor-pointer outline-none xl:w-full';

export function AppNav({ unreadCount = 0 }: { unreadCount?: number }) {
  const pathname = usePathname();
  const { user, isLoggedIn, logout } = useAuth();
  const openSignInModal = useUIStore((s) => s.openSignInModal);
  const [composing, setComposing] = useState(false);
  const [confirmLogout, setConfirmLogout] = useState(false);

  const profileHref = user ? `/${user.username}` : null;
  const items: { label: string; icon: LucideIcon; href: string | null; badge?: number; match?: (path: string) => boolean }[] = [
    { label: 'Home', icon: Home, href: '/' },
    { label: 'Explore', icon: Search, href: '/explore', match: (p) => p === '/explore' || p === '/search' },
    { label: 'Notifications', icon: Bell, href: user ? '/notifications' : null, badge: unreadCount },
    { label: 'Messages', icon: Mail, href: '/messages' },
    { label: 'ChanomBot', icon: BotMessageSquare, href: '/chanombot' },
    { label: 'Connect', icon: Users, href: '/connect_people' },
    // A profile's tabs and follower pages all count as "Profile".
    { label: 'Profile', icon: User, href: profileHref, match: (p) => !!profileHref && (p === profileHref || p.startsWith(`${profileHref}/`)) },
  ];

  return (
    <nav className="no-scrollbar sticky top-0 flex h-screen w-full flex-col items-end justify-between overflow-y-auto p-2 xl:items-start xl:p-4" aria-label="Primary">
      <div className="flex w-full flex-col items-end gap-1 xl:items-start">
        <Link href="/" aria-label="Home" className="mb-1 flex size-[52px] items-center justify-center rounded-full transition-colors hover:bg-accent">
          <XLogo className="size-8 text-foreground" />
        </Link>

        {items.map((item) => {
          const active = item.match ? item.match(pathname) : pathname === item.href;
          const content = <NavItemContent icon={item.icon} label={item.label} active={active} badge={item.badge} />;
          return item.href ? (
            <Link key={item.label} href={item.href} aria-current={active ? 'page' : undefined} className={itemClass}>
              {content}
            </Link>
          ) : (
            // Signed out: these need an account.
            <button key={item.label} type="button" onClick={openSignInModal} className={itemClass}>
              {content}
            </button>
          );
        })}

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button type="button" className={itemClass}>
              <NavItemContent icon={MoreHorizontal} label="More" active={pathname === '/premium' || pathname === '/brainrot'} />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="top" align="start" className="w-[220px] rounded-xl py-2 font-bold">
            <DropdownMenuItem asChild className="cursor-pointer gap-4 p-3 text-[15px]">
              <Link href="/premium">
                <XLogo className="size-5" />
                Premium
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild className="cursor-pointer gap-4 p-3 text-[15px]">
              <Link href="/brainrot">
                <TvMinimalPlay className="size-5" />
                Brainrot
              </Link>
            </DropdownMenuItem>
            <ThemeToggle variant="menu-item" />
          </DropdownMenuContent>
        </DropdownMenu>

        {isLoggedIn && (
          <button
            type="button"
            onClick={() => setComposing(true)}
            aria-label="Post"
            className="mt-4 flex size-[52px] cursor-pointer items-center justify-center rounded-full bg-primary text-[17px] font-bold text-primary-foreground shadow-lg transition-opacity hover:opacity-90 xl:h-[52px] xl:w-[90%]"
          >
            <span className="hidden xl:inline">Post</span>
            <svg viewBox="0 0 24 24" aria-hidden="true" className="size-6 fill-current xl:hidden">
              <path d="M23 3c-6.62-.1-10.38 2.421-13.05 6.03C7.29 12.61 6 17.331 6 22h2c0-1.007.07-2.012.19-3H12c4.1 0 7.48-3.082 7.94-7.054C22.79 10.147 23.17 6.359 23 3zm-7 8h-1.5v2H16c.63-.016 1.2-.08 1.72-.188C16.95 15.24 14.68 17 12 17H8.55c.57-2.512 1.57-4.851 3-6.78 2.16-2.912 5.29-4.911 9.45-5.187C20.95 8.079 19.9 11 16 11zM4 9V6H1V4h3V1h2v3h3v2H6v3H4z" />
            </svg>
          </button>
        )}
      </div>

      <div className="mb-3 flex w-full justify-end xl:justify-start">
        {user ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button type="button" className="flex w-fit cursor-pointer items-center gap-3 rounded-full p-3 text-left outline-none transition-colors hover:bg-accent xl:w-full" aria-label="Account menu">
                <UserAvatar user={user} />
                <span className="hidden min-w-0 flex-1 xl:block">
                  <span className="block truncate text-[15px] font-bold leading-5">{displayNameOf(user)}</span>
                  <span className="block truncate text-[15px] leading-5 text-muted-foreground">@{user.username}</span>
                </span>
                <MoreHorizontal className="hidden size-5 xl:block" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="top" align="center" className="w-[300px] rounded-xl py-2 font-bold">
              <DropdownMenuItem className="cursor-pointer p-3 text-[15px]" onSelect={() => setConfirmLogout(true)}>
                Log out @{user.username}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <button
            type="button"
            onClick={openSignInModal}
            aria-label="Sign in"
            className="flex size-[52px] cursor-pointer items-center justify-center rounded-full bg-primary font-bold text-primary-foreground hover:opacity-90 xl:h-12 xl:w-full"
          >
            <span className="hidden text-[15px] xl:inline">Sign in</span>
            <LogIn className="size-6 xl:hidden" />
          </button>
        )}
      </div>

      <ComposeTweetModal isOpen={composing} onClose={() => setComposing(false)} />
      <ConfirmModal
        open={confirmLogout}
        title="Log out of Chanom Twitter?"
        description="You can always log back in at any time."
        confirmLabel="Log out"
        onConfirm={() => {
          setConfirmLogout(false);
          logout();
        }}
        onClose={() => setConfirmLogout(false)}
      />
    </nav>
  );
}
