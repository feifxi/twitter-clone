'use client';

import { AppNav } from '@/components/AppNav';
import { Sidebar } from '@/components/Sidebar';
import { MobileNav } from '@/components/MobileNav';

import { usePathname } from 'next/navigation';
import { useUnreadCount } from '@/hooks/useNotifications';
import { useRealtimeEvents } from '@/hooks/useRealtimeEvents';

/** Navigation, sidebar and the realtime stream around every main page. */
export function MainShell({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();

  const isMessagesPage = pathname?.startsWith('/messages');
  const isChanombotPage = pathname?.startsWith('/chanombot');
  const isWideLayout = isMessagesPage || isChanombotPage;
  const { data: unreadCount } = useUnreadCount();

  // One SSE stream per tab, mounted at layout scope (notifications + direct messages).
  useRealtimeEvents();

  return (
    <div
      className="min-h-screen bg-background text-foreground flex justify-center"
      style={{ fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, sans-serif' }}
    >
      {/* Left gutter: flex grow so content is centered, then fixed columns */}
      <div className={`flex justify-center flex-1 min-w-0 ${isWideLayout ? 'max-w-[1500px]' : 'max-w-[1280px]'}`}>
        {/* Left Sidebar */}
        <aside className="hidden sm:flex w-[68px] xl:w-[275px] shrink-0 justify-end">
          <AppNav unreadCount={unreadCount ?? 0} />
        </aside>
        {/* Main Feed */}
        <main className={`w-full border-x border-border min-h-screen pb-[60px] sm:pb-0 ${isWideLayout ? 'max-w-[1000px] flex-1' : 'max-w-[600px]'}`}>
          {children}
        </main>
        {/* Right Sidebar: hidden on smaller screens, and hidden on messages page */}
        {!isWideLayout && (
          <aside className="w-[350px] shrink-0 hidden lg:block">
            <Sidebar />
          </aside>
        )}
      </div>
      <MobileNav unreadCount={unreadCount ?? 0} />
    </div>
  );
}
