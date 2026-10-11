'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Home, Search, Bell, Mail, Users, LogIn } from 'lucide-react';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { ComposeTweetModal } from '@/components/ComposeTweetModal';
import { useUIStore } from '@/store/useUIStore';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

import { FILL_WHEN_ACTIVE } from '@/components/AppNav';
import { ConfirmModal } from '@/components/ConfirmModal';
import { ThemeToggle } from '@/components/ThemeToggle';
import { UserAvatar } from '@/components/UserAvatar';

type MobileNavProps = {
  unreadCount?: number;
};

export function MobileNav({ unreadCount = 0 }: MobileNavProps) {
  const pathname = usePathname();
  const { user, isLoggedIn, logout } = useAuth();
  const [showCompose, setShowCompose] = useState(false);
  const [confirmLogout, setConfirmLogout] = useState(false);
  const openSignInModal = useUIStore((s) => s.openSignInModal);

  const NAV_ITEMS = [
    { label: 'Home', icon: Home, href: '/' },
    { label: 'Explore', icon: Search, href: '/explore' },
    { label: 'Connect', icon: Users, href: '/connect_people' },
    { label: 'Notifications', icon: Bell, href: user ? '/notifications' : '#', onClick: user ? undefined : openSignInModal },
    { label: 'Messages', icon: Mail, href: '/messages' },
  ];

  return (
    <div className="sm:hidden block">
      {/* Floating Action Buttons */}
      <div className="fixed bottom-20 right-4 flex flex-col gap-4 z-40">
        {/* More/Profile Actions Menu that floats above the tweet button on mobile if needed, or we just put it in the nav */}
        {isLoggedIn && (
          <Button
            aria-label="Post"
            className="w-[56px] h-[56px] rounded-full flex items-center justify-center shadow-[0_4px_14px_rgba(0,0,0,0.25)] p-0 bg-primary hover:bg-primary/90 transition-colors"
            size="icon"
            onClick={() => setShowCompose(true)}
          >
            <svg viewBox="0 0 24 24" aria-hidden="true" className="w-6 h-6 fill-primary-foreground"><g><path d="M23 3c-6.62-.1-10.38 2.421-13.05 6.03C7.29 12.61 6 17.331 6 22h2c0-1.007.07-2.012.19-3H12c4.1 0 7.48-3.082 7.94-7.054C22.79 10.147 23.17 6.359 23 3zm-7 8h-1.5v2H16c.63-.016 1.2-.08 1.72-.188C16.95 15.24 14.68 17 12 17H8.55c.57-2.512 1.57-4.851 3-6.78 2.16-2.912 5.29-4.911 9.45-5.187C20.95 8.079 19.9 11 16 11zM4 9V6H1V4h3V1h2v3h3v2H6v3H4z"></path></g></svg>
          </Button>
        )}
      </div>

      {/* Fixed Bottom Bar */}
      <nav className="fixed bottom-0 left-0 right-0 bg-background/90 backdrop-blur-md border-t border-border z-50 px-2 h-14 flex items-center justify-around pb-safe">
        {NAV_ITEMS.map((item) => {
          const isActive = pathname === item.href;
          const Icon = item.icon;

          if (item.onClick) {
            return (
              <button
                key={item.label}
                type="button"
                aria-label={item.label}
                onClick={item.onClick}
                className="p-2 relative flex items-center justify-center cursor-pointer"
              >
                <Icon className={`w-6 h-6 ${isActive ? 'text-foreground' : 'text-muted-foreground'} ${isActive && FILL_WHEN_ACTIVE.has(Icon) ? 'fill-current' : ''}`} strokeWidth={isActive ? 2.5 : 2} />
                {item.label === 'Notifications' && unreadCount > 0 && (
                  <div className="absolute top-1 right-1 w-2.5 h-2.5 rounded-full bg-primary border-2 border-background" />
                )}
              </button>
            );
          }

          return (
            <Link
              key={item.label}
              href={item.href}
              aria-label={item.label}
              aria-current={isActive ? 'page' : undefined}
              className="p-2 relative flex items-center justify-center cursor-pointer"
            >
              <Icon className={`w-6 h-6 ${isActive ? 'text-foreground' : 'text-muted-foreground'} ${isActive && FILL_WHEN_ACTIVE.has(Icon) ? 'fill-current' : ''}`} strokeWidth={isActive ? 2.5 : 2} />
              {item.label === 'Notifications' && unreadCount > 0 && (
                <div className="absolute top-1 right-1 w-2.5 h-2.5 rounded-full bg-primary border-2 border-background" />
              )}
            </Link>
          );
        })}

        {/* Profile / Menu as the last item */}
        {isLoggedIn && user ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button type="button" aria-label="Account menu" className="p-2 relative flex items-center justify-center cursor-pointer outline-none">
                <UserAvatar user={user} size="xs" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" side="top" className="w-[200px] mb-2 font-bold z-50">
              <DropdownMenuItem asChild className="p-3 text-[15px] cursor-pointer">
                <Link href={`/${user.username}`}>Profile</Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild className="p-3 text-[15px] cursor-pointer">
                <Link href="/chanombot">ChanomBot</Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild className="p-3 text-[15px] cursor-pointer">
                <Link href="/premium">Premium</Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild className="p-3 text-[15px] cursor-pointer">
                <Link href="/brainrot">Brainrot</Link>
              </DropdownMenuItem>
              <ThemeToggle variant="menu-item" />
              <DropdownMenuItem
                className="p-3 text-[15px] cursor-pointer text-destructive focus:text-destructive"
                onSelect={() => setConfirmLogout(true)}
              >
                Log out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <button type="button" aria-label="Sign in" className="p-2 relative flex items-center justify-center cursor-pointer" onClick={openSignInModal}>
            <LogIn className="w-6 h-6 text-muted-foreground" strokeWidth={2} />
          </button>
        )}
      </nav>

      {/* Global Compose Modal */}
      {isLoggedIn && <ComposeTweetModal isOpen={showCompose} onClose={() => setShowCompose(false)} />}
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
    </div>
  );
}
