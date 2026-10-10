'use client';

import Link from 'next/link';
import { useRouter, usePathname } from 'next/navigation'; // Added usePathname
import { useState, useRef, useEffect } from 'react';
import { Search } from 'lucide-react';
import { useTrendingHashtags, useSuggestedUsers, useSearchUsers, useSearchHashtags } from '@/hooks/useDiscovery';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { useDebounce } from '@/hooks/useDebounce';
import { FollowButton } from '@/components/FollowButton';
import type { HashtagResponse, UserResponse, PageResponse } from '@/types';
import { UserAvatar } from '@/components/UserAvatar';

export function Sidebar() {
  const router = useRouter();
  const pathname = usePathname();
  const [query, setQuery] = useState('');
  const [isFocused, setIsFocused] = useState(false);
  const debouncedQuery = useDebounce(query, 500);
  const searchRef = useRef<HTMLDivElement>(null);

  const { data: searchResults, isLoading: isSearchLoading } = useSearchUsers(debouncedQuery);
  const { data: hashtagResults, isLoading: isHashtagLoading } = useSearchHashtags(debouncedQuery);
  const { data: trending, isLoading: trendingLoading } = useTrendingHashtags(4);
  const { data: suggested, isLoading: suggestedLoading } = useSuggestedUsers(3);
  const { user: currentUser } = useAuth();

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (searchRef.current && !searchRef.current.contains(event.target as Node)) {
        setIsFocused(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [searchRef]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      router.push(`/search?q=${encodeURIComponent(query)}`);
      setIsFocused(false);
      setQuery(''); // Clear input after search
    }
  };

  const subscribeToPremium = () => {
    router.push('/premium');
  };



  // If on search page, hide search bar but show other widgets
  if (pathname === '/search' || pathname?.startsWith('/search')) {
    return (
      <div className="sticky top-0 p-4 h-screen overflow-y-auto no-scrollbar pb-10">
        <PremiumCard subscribeToPremium={subscribeToPremium} />
        <TrendingList trending={trending} trendingLoading={trendingLoading} />
        <SuggestedUsersList suggested={suggested} suggestedLoading={suggestedLoading} currentUser={currentUser} />
        <Footer />
      </div>
    );
  }

  return (
    <div className="sticky top-0 p-4 py-2 h-screen overflow-y-auto no-scrollbar pb-10">
      {/* Top Bar with Search */}
      <div className="mb-4 sticky top-0 bg-background z-20 py-1 flex items-center gap-2" ref={searchRef}>
        <form onSubmit={handleSearch} className="relative flex-1">
          <div className={`flex items-center gap-3 rounded-full bg-secondary/50 border px-4 py-2.5 transition-colors ${
            isFocused ? 'bg-background border-primary' : 'border-border'
          }`}>
            <Search className={`w-5 h-5 shrink-0 ${isFocused ? 'text-primary' : 'text-muted-foreground'}`} />
            <input
              type="search"
              placeholder="Search"
              className="flex-1 min-w-0 bg-transparent text-foreground text-[15px] placeholder:text-muted-foreground outline-none border-none leading-5"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onFocus={() => setIsFocused(true)}
            />
          </div>

          {/* Search Dropdown */}
          {isFocused && query && (
            <div className="absolute top-full left-0 right-0 mt-1 bg-background border border-border rounded-lg shadow-xl overflow-hidden min-h-[100px] z-50 max-h-[400px] overflow-y-auto">
              {isSearchLoading || isHashtagLoading ? (
                <div className="p-4 text-center text-muted-foreground">Loading...</div>
              ) : (
                <ul>
                  {searchResults?.items.length === 0 && hashtagResults?.length === 0 && (
                    <li className="p-4 text-muted-foreground text-center">No results found</li>
                  )}

                  {/* People Results */}
                  {searchResults && searchResults.items.length > 0 && (
                    <li className="px-4 py-2 text-sm text-muted-foreground font-bold bg-secondary/30">People</li>
                  )}
                  {searchResults?.items.map(u => (
                    <li key={u.id}>
                      <Link
                        href={`/${u.username}`}
                        className="flex items-center gap-3 px-4 py-3 hover:bg-card transition-colors"
                        onClick={() => setIsFocused(false)}
                      >
                        <UserAvatar user={u} />
                        <div className="flex-1 min-w-0">
                          <div className="font-bold text-foreground truncate">{u.displayName}</div>
                          <div className="text-muted-foreground truncate">@{u.username}</div>
                        </div>
                      </Link>
                    </li>
                  ))}

                  {/* Hashtag Results */}
                  {hashtagResults && hashtagResults.length > 0 && (
                    <li className="px-4 py-2 text-sm text-muted-foreground font-bold bg-secondary/30 border-t border-border">Hashtags</li>
                  )}
                  {hashtagResults?.map((h: HashtagResponse) => (
                    <li key={h.text}>
                      <Link
                        href={`/search?q=${encodeURIComponent('#' + h.text)}`}
                        className="flex items-center gap-3 px-4 py-3 hover:bg-card transition-colors"
                        onClick={() => setIsFocused(false)}
                      >
                        <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                          <Search className="w-5 h-5 text-primary" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="font-bold text-foreground truncate">#{h.text}</div>
                          <div className="text-muted-foreground truncate">{h.usageCount} posts</div>
                        </div>
                      </Link>
                    </li>
                  ))}

                  <li className="border-t border-border">
                    <Link
                      href={`/search?q=${encodeURIComponent(query)}`}
                      className="block px-4 py-3 text-primary hover:bg-card transition-colors"
                      onClick={() => setIsFocused(false)}
                    >
                      Search for &quot;{query}&quot;
                    </Link>
                  </li>
                </ul>
              )}
            </div>
          )}
        </form>
      </div>

      <PremiumCard subscribeToPremium={subscribeToPremium} />
      <TrendingList trending={trending} trendingLoading={trendingLoading} />
      <SuggestedUsersList suggested={suggested} suggestedLoading={suggestedLoading} currentUser={currentUser} />
      <Footer />
    </div>
  );
}

// Helper Components
function Footer() {
  return (
    <div className="px-4 text-[13px] text-muted-foreground space-x-2 leading-4">
      <span>Terms of Service</span>
      <span>Privacy Policy</span>
      <span>Cookie Policy</span>
      <span>Accessibility</span>
      <span>Ads info</span>
      <span>More ...</span>
      <p className="mt-1">© 2026 Chanombude Corp.</p>
    </div>
  );
}

function MoreHorizontal(props: React.SVGProps<SVGSVGElement>) {
  return <svg {...props} xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>
}

interface SuggestedUsersListProps {
  suggested: PageResponse<UserResponse> | undefined;
  suggestedLoading: boolean;
  currentUser: UserResponse | null;
}

function SuggestedUsersList({ suggested, suggestedLoading, currentUser }: SuggestedUsersListProps) {
  return (
    <div className="rounded-2xl border border-border bg-card overflow-hidden pt-3 mb-20">
      <h2 className="px-4 pb-2 font-extrabold text-[20px]">
        Who to follow
      </h2>
      {suggestedLoading ? (
        <div className="px-4 pb-3 space-y-4 pt-2">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-12 rounded-lg bg-secondary/50 animate-pulse"
            />
          ))}
        </div>
      ) : (
        <div className="px-0">
          {suggested?.items
            ?.filter((u: UserResponse) => u.id !== currentUser?.id)
            .slice(0, 3)
            .map((u: UserResponse) => (
              <div
                key={u.id}
                className="flex items-center gap-3 px-4 py-3 hover:bg-secondary/30 transition-colors cursor-pointer"
              >
                <Link href={`/${u.username}`} className="shrink-0">
                  <UserAvatar user={u} />
                </Link>
                <div className="flex-1 min-w-0">
                  <Link
                    href={`/${u.username}`}
                    className="font-bold text-foreground hover:underline block truncate text-[15px] leading-5"
                  >
                    {u.displayName}
                  </Link>
                  <p className="text-muted-foreground text-[15px] truncate leading-5">
                    @{u.username}
                  </p>
                </div>
                <FollowButton userId={u.id} isFollowing={u.isFollowing} />
              </div>
            ))}
          <Link href="/connect_people" className="block px-4 py-3 text-primary text-[15px] hover:bg-secondary/30 transition-colors rounded-b-2xl">
            Show more
          </Link>
        </div>
      )}
    </div>
  );
}

function PremiumCard({ subscribeToPremium }: { subscribeToPremium: () => void }) {
  return (
    <div className="rounded-2xl border border-border bg-card mb-4 p-4 mt-2">
      <h2 className="font-bold text-xl mb-2">Subscribe to Premium</h2>
      <p className="text-[15px] text-muted-foreground mb-3 leading-5">Subscribe to unlock new features and if eligible, receive a share of ads revenue.</p>
      <Button className="rounded-full font-bold px-4 cursor-pointer" size="sm" onClick={subscribeToPremium}>Subscribe</Button>
    </div>
  );
}

interface TrendingListProps {
  trending: HashtagResponse[] | undefined;
  trendingLoading: boolean;
}

function TrendingList({ trending, trendingLoading }: TrendingListProps) {
  return (
    <div className="rounded-2xl border border-border bg-card mb-4 overflow-hidden pt-3">
      <h2 className="px-4 pb-2 font-extrabold text-[20px]">
        What&apos;s happening
      </h2>
      {trendingLoading ? (
        <div className="px-4 pb-3 space-y-4 pt-2">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-10 rounded-lg bg-secondary/50 animate-pulse"
            />
          ))}
        </div>
      ) : (
        <div className="px-0">
          {trending?.slice(0, 5).map((item: HashtagResponse) => (
            <Link
              key={item.text}
              href={`/search?q=${encodeURIComponent('#' + item.text)}`}
              className="block px-4 py-3 hover:bg-secondary/30 transition-colors"
            >
              <div className="flex justify-between items-start">
                <div>
                  <p className="text-muted-foreground text-[13px] leading-4">Trending in generic</p>
                  <p className="text-foreground font-bold text-[15px] pt-0.5">
                    #{item.text}
                  </p>
                  <p className="text-muted-foreground text-[13px] leading-4 pt-0.5">
                    {item.usageCount} posts
                  </p>
                </div>
                <MoreHorizontal className="w-4 h-4 text-muted-foreground" />
              </div>
            </Link>
          ))}
          <Link href="/explore" className="block px-4 py-3 text-primary text-[15px] hover:bg-secondary/30 transition-colors rounded-b-2xl">
            Show more
          </Link>
        </div>
      )}
    </div>
  );
}
