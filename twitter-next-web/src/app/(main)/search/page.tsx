'use client';

import { useSearchParams, useRouter, usePathname } from 'next/navigation';
import { Suspense, useState, useEffect } from 'react';
import { Search, ArrowLeft } from 'lucide-react';
import { FeedList } from '@/components/FeedList';
import { useQuery } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import type { PageResponse, TweetResponse } from '@/types';


  import { useSearchUsers } from '@/hooks/useDiscovery';
  import { queryKeys } from '@/lib/query-keys';
  import Link from 'next/link';
  import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
  import { FollowButton } from '@/components/FollowButton';
  import { useAuth } from '@/hooks/useAuth';

  function SearchContent() {
  const searchParams = useSearchParams();
  const q = searchParams.get('q') ?? '';
  const pathname = usePathname();
  const [inputValue, setInputValue] = useState(q);
  const activeTab = searchParams.get('tab') === 'people' ? 'people' : 'tweets';
  const router = useRouter(); 
  
  const setActiveTab = (tab: 'tweets' | 'people') => {
    if (tab !== activeTab) {
        const params = new URLSearchParams(searchParams);
        params.set('tab', tab);
        router.replace(`${pathname}?${params.toString()}`);
    }
  }; 

  // Sync local input with URL param changes (e.g. navigation)
  useEffect(() => {
    setInputValue(q);
  }, [q]);

  // Tweets Query - uses 'q' from URL, not local state
  const { data: tweetData, isLoading: isTweetLoading } = useQuery({
    queryKey: queryKeys.search.tweets(q),
    queryFn: async (): Promise<PageResponse<TweetResponse>> => {
      const { data: res } = await axiosInstance.get<PageResponse<TweetResponse>>(
        '/search/tweets',
        { params: { q, size: 20 } }
      );
      return res;
    },
    enabled: q.length > 0 && activeTab === 'tweets',
  });

  // People Query - uses 'q' from URL
  const { data: userData, isLoading: isUserLoading } = useSearchUsers(q, activeTab === 'people');
  
  const tweets = tweetData?.items ?? [];
  const users = userData?.items ?? [];

  const { user: currentUser } = useAuth();

  const handleSearch = (e: React.FormEvent) => {
      e.preventDefault();
      // Update URL to reflect search - this triggers the queries
      if (inputValue.trim()) {
        router.push(`/search?q=${encodeURIComponent(inputValue)}`);
      }
  };

  return (
    <div className="min-h-screen">
      <div className="sticky top-0 z-30 bg-background/60 backdrop-blur-md border-b border-border px-4 py-3">
        <div className="flex items-center gap-4">
            <div 
              onClick={() => router.back()} 
              className="cursor-pointer p-2 rounded-full hover:bg-card transition-colors -ml-2"
            >
              <ArrowLeft className="w-5 h-5 text-foreground" />
            </div>
        <form onSubmit={handleSearch} className="flex-1">
            <div className="flex items-center gap-2 rounded-full bg-secondary/50 border border-border focus-within:border-primary focus-within:bg-background px-4 py-2.5 transition-colors">
            <Search className="w-5 h-5 text-muted-foreground shrink-0 focus-within:text-primary" />
            <input
                type="search"
                placeholder="Search"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                className="flex-1 min-w-0 bg-transparent text-foreground text-[15px] placeholder:text-muted-foreground outline-none border-none"
            />
            </div>
        </form>
      </div>
        
        {/* Tabs */}
        <div className="flex mt-3 border-b border-border">
            {/* Tweets Tab */}
            <button
                onClick={() => setActiveTab('tweets')}
                className={`cursor-pointer flex-1 py-4 text-[15px] font-bold relative hover:bg-card transition-colors ${activeTab === 'tweets' ? 'text-foreground' : 'text-muted-foreground'}`}
            >
                Top
                {activeTab === 'tweets' && (
                    <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-14 h-1 bg-primary rounded-full" />
                )}
            </button>
            {/* People Tab */}
             <button
                onClick={() => setActiveTab('people')}
                className={`cursor-pointer flex-1 py-4 text-[15px] font-bold relative hover:bg-card transition-colors ${activeTab === 'people' ? 'text-foreground' : 'text-muted-foreground'}`}
            >
                People
                {activeTab === 'people' && (
                    <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-14 h-1 bg-primary rounded-full" />
                )}
            </button>
        </div>
      </div>

      {q.length === 0 ? (
        <p className="p-4 text-muted-foreground text-[15px]">
          Enter a search term.
        </p>
      ) : (
        <>
            {activeTab === 'tweets' ? (
                <FeedList tweets={tweets} isLoading={isTweetLoading} />
            ) : (
                <div className="flex flex-col">
                    {isUserLoading ? (
                         <div className="p-4 text-center text-muted-foreground">Loading...</div>
                    ) : users.length === 0 ? (
                        <div className="p-4 text-center text-muted-foreground">No people found for &quot;{q}&quot;</div>
                    ) : (
                        users.map((u) => (
                            <div key={u.id} className="flex items-center gap-3 px-4 py-3 hover:bg-card transition-colors cursor-pointer border-b border-border">
                                 <Link href={`/${u.username}`} className="shrink-0">
                                    <Avatar className="w-10 h-10">
                                        <AvatarImage src={u.avatarUrl ?? undefined} />
                                        <AvatarFallback>{(u.displayName || u.username)[0]}</AvatarFallback>
                                    </Avatar>
                                </Link>
                                <div className="flex-1 min-w-0">
                                    <Link href={`/${u.username}`} className="font-bold text-foreground hover:underline block truncate">
                                        {u.displayName}
                                    </Link>
                                    <div className="text-muted-foreground truncate">@{u.username}</div>
                                    {u.bio && <p className="text-foreground text-[14px] mt-1">{u.bio}</p>}
                                </div>
                                {currentUser?.id !== u.id && (
                                    <FollowButton userId={u.id} isFollowing={u.isFollowing} />
                                )}
                            </div>
                        ))
                    )}
                </div>
            )}
        </>
      )}
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense
      fallback={
        <div className="p-4 text-muted-foreground text-[15px]">Loading...</div>
      }
    >
      <SearchContent />
    </Suspense>
  );
}
