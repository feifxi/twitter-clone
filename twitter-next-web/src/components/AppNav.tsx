'use client';

import { useState } from 'react';

import Link from 'next/link';
import { Home, User, MoreHorizontal, Bell, Mail, Search, Users, BotMessageSquare, TvMinimalPlay, LogIn } from 'lucide-react';
import { usePathname } from 'next/navigation';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { XLogo } from '@/components/XLogo';
import { ComposeTweetModal } from '@/components/ComposeTweetModal';
import { useUIStore } from '@/store/useUIStore';
import { ThemeToggle } from '@/components/ThemeToggle';
import { UserAvatar } from '@/components/UserAvatar';

type AppNavProps = {
  unreadCount?: number;
};

export function AppNav({ unreadCount = 0 }: AppNavProps) {
  const pathname = usePathname();
  const { user, isLoggedIn, logout } = useAuth();
  const [showCompose, setShowCompose] = useState(false);
  const openSignInModal = useUIStore((s) => s.openSignInModal);

  const NAV_ITEMS = [
    { label: 'Home', icon: Home, href: '/' },
    { label: 'Explore', icon: Search, href: '/explore' },
    { label: 'Notifications', icon: Bell, href: user ? '/notifications' : '#', onClick: user ? undefined : openSignInModal },
    { label: 'Messages', icon: Mail, href: '/messages' },
    { label: 'ChanomBot', icon: BotMessageSquare, href: '/chanombot' },
    { label: 'Connect', icon: Users, href: '/connect_people' },
    { label: 'Profile', icon: User, href: user ? `/${user.username}` : '#', onClick: user ? undefined : openSignInModal },
    { label: 'More', icon: MoreHorizontal, href: '/brainrot' },
  ];

  return (
    <nav className="flex flex-col items-end xl:items-start w-full h-screen p-2 xl:p-4 sticky top-0 justify-between overflow-y-auto no-scrollbar">
      <div className="flex flex-col gap-1 w-full">
        {/* Logo */}
        <Link
          href="/"
          className="size-12 flex items-center justify-center rounded-full hover:bg-accent/50 transition-colors mb-1 xl:ml-0"
        >
          <XLogo className="w-7 h-7 fill-foreground" />
        </Link>

        {/* Navigation Items */}
        {NAV_ITEMS.map((item) => {
          const isActive = pathname === item.href;
          
          if (item.onClick) {
             return (
                <button
                key={item.label}
                type="button"
                onClick={item.onClick}
                className="group flex items-center w-fit xl:w-full"
                >
                <div
                    className={`flex items-center xl:gap-5 px-3 py-3 rounded-full transition-colors group-hover:bg-accent/50 ${
                    isActive ? 'font-bold' : 'font-normal'
                    }`}
                >
                    <item.icon
                    className={`w-7 h-7 ${
                        isActive ? 'fill-current' : ''
                    }`}
                    strokeWidth={isActive ? 2.5 : 2}
                    />
                    <div className="relative hidden xl:block cursor-pointer">
                        <span className="text-[20px] mr-4 leading-6">{item.label}</span>
                         {item.label === 'Notifications' && unreadCount > 0 && (
                            <div className="absolute -top-1 -right-1 xl:top-0 xl:-right-2 w-4 h-4 rounded-full bg-primary flex items-center justify-center text-[10px] text-foreground font-bold">
                                {unreadCount}
                            </div>
                        )}
                    </div>
                </div>
                </button>
             )
          }

          if (item.label === 'More') {
             return (
                 <DropdownMenu key={item.label}>
                    <DropdownMenuTrigger asChild>
                        <button type="button" className="group flex items-center w-fit xl:w-full outline-none cursor-pointer">
                            <div className={`flex items-center xl:gap-5 px-3 py-3 rounded-full transition-colors group-hover:bg-accent/50 ${isActive ? 'font-bold' : 'font-normal'}`}>
                                <item.icon className={`w-7 h-7 ${isActive ? 'fill-current' : ''}`} strokeWidth={isActive ? 2.5 : 2} />
                                <div className="relative hidden xl:block cursor-pointer">
                                    <span className="text-[20px] mr-4 leading-6">{item.label}</span>
                                </div>
                            </div>
                        </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent className="w-[200px] rounded-xl border border-border shadow-lg py-2 bg-background font-bold ml-10 xl:ml-0" align="start" side="top">
                        <DropdownMenuItem asChild className="p-3 text-[15px] cursor-pointer focus:bg-accent">
                            <Link href="/premium" className="flex items-center gap-4">
                                <XLogo className="w-5 h-5 fill-current" />
                                <span>Premium</span>
                            </Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem asChild className="p-3 text-[15px] cursor-pointer focus:bg-accent">
                            <Link href="/brainrot" className="flex items-center gap-4">
                                <TvMinimalPlay className="w-5 h-5" />
                                <span>Brainrot</span>
                            </Link>
                        </DropdownMenuItem>
                        <ThemeToggle variant="menu-item" />
                    </DropdownMenuContent>
                 </DropdownMenu>
             )
          }

          return (
            <Link
              key={item.label}
              href={item.href}
              className="group flex items-center w-fit xl:w-full"
            >
              <div
                className={`flex items-center xl:gap-5 px-3 py-3 rounded-full transition-colors group-hover:bg-accent/50 ${
                  isActive ? 'font-bold' : 'font-normal'
                }`}
              >
                <item.icon
                  className={`w-7 h-7 ${
                    isActive ? 'fill-current' : ''
                  }`}
                  strokeWidth={isActive ? 2.5 : 2}
                />
                <div className="relative">
                    <span className="text-[20px] mr-4 leading-6 hidden xl:block">{item.label}</span>
                    {item.label === 'Notifications' && unreadCount > 0 && (
                        <div className="absolute -top-1 -right-1 xl:top-0 xl:-right-2 w-4 h-4 rounded-full bg-primary flex items-center justify-center text-[10px] text-foreground font-bold">
                            {unreadCount}
                        </div>
                    )}
                </div>
              </div>
            </Link>
          );
        })}

        {/* Tweet Button (Large) */}
        {isLoggedIn && (
          <Button
            className="hidden xl:block mt-4 rounded-full text-[17px] font-bold shadow-lg cursor-pointer"
            size="lg"
            onClick={() => setShowCompose(true)}
          >
            Post
          </Button>
        )}
        {/* Mobile Tweet Button (Circle) */}
         {isLoggedIn && (
          <Button
            className="xl:hidden w-[50px] h-[50px] mt-4 rounded-full flex items-center justify-center shadow-lg p-0"
            size="icon"
            onClick={() => setShowCompose(true)}
          >
            <svg viewBox="0 0 24 24" aria-hidden="true" className="w-6 h-6 fill-white"><g><path d="M23 3c-6.62-.1-10.38 2.421-13.05 6.03C7.29 12.61 6 17.331 6 22h2c0-1.007.07-2.012.19-3H12c4.1 0 7.48-3.082 7.94-7.054C22.79 10.147 23.17 6.359 23 3zm-7 8h-1.5v2H16c.63-.016 1.2-.08 1.72-.188C16.95 15.24 14.68 17 12 17H8.55c.57-2.512 1.57-4.851 3-6.78 2.16-2.912 5.29-4.911 9.45-5.187C20.95 8.079 19.9 11 16 11zM4 9V6H1V4h3V1h2v3h3v2H6v3H4z"></path></g></svg>
          </Button>
        )}
        
        <ComposeTweetModal isOpen={showCompose} onClose={() => setShowCompose(false)} />
      </div>

      {/* User Profile / Logout */}
      <div className="flex flex-col items-center xl:items-start w-full gap-2">
        {isLoggedIn && user ? (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="flex items-center justify-center xl:justify-between w-fit xl:w-full h-auto p-3 rounded-full hover:bg-accent/50 mb-4 cursor-pointer"
            >
              <div className="flex items-center gap-3 truncate">
                <UserAvatar user={user} />
                <div className="hidden xl:flex flex-col items-start min-w-0">
                  <span className="font-bold text-[15px] truncate max-w-[140px] leading-5">
                    {user.displayName}
                  </span>
                  <span className="text-muted-foreground text-[15px] truncate max-w-[140px] leading-5">
                    @{user.username}
                  </span>
                </div>
              </div>
              <MoreHorizontal className="hidden xl:block w-5 h-5 text-muted-foreground" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[300px] mb-2 rounded-xl border border-border shadow-lg py-2 bg-background font-bold"
            side="top"
            align="center"
          >
             <DropdownMenuItem 
                className="p-3 text-[15px] cursor-pointer focus:bg-accent"
                onClick={() => {
                  logout();
                  openSignInModal();
                }}
             >
                Switch account
             </DropdownMenuItem>
            <DropdownMenuItem
              className="p-3 text-[15px] cursor-pointer focus:bg-accent"
              onClick={() => logout()}
            >
              Log out @{user.username}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : (
          <div className="w-full xl:w-[240px] mb-4 flex justify-end xl:justify-start">
              {/* Desktop Sign In Button */}
              <Button 
                  className="hidden xl:block w-full rounded-full font-bold h-[48px] text-[15px] cursor-pointer" 
                  onClick={openSignInModal}
              >
                  Sign in
              </Button>
              {/* Mobile Sign In Button (Circle) */}
               <Button
                  className="xl:hidden w-[50px] h-[50px] rounded-full flex items-center justify-center shadow-lg p-0 cursor-pointer"
                  onClick={openSignInModal}
              >
                  <LogIn className="w-6 h-6" />
              </Button>
          </div>
        )}
      </div>
    </nav>
  );
}
