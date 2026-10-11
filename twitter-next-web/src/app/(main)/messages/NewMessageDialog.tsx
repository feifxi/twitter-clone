'use client';

import { useState } from 'react';
import { Search } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';
import { UserAvatar } from '@/components/UserAvatar';
import { useDebounce } from '@/hooks/useDebounce';
import { useSearchUsers, useSuggestedUsers } from '@/hooks/useDiscovery';
import { displayNameOf } from '@/lib/format';
import type { UserResponse } from '@/types';

interface NewMessageDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPick: (user: UserResponse) => void;
  myId?: number;
}

/** Pick someone to message: suggestions first, search results once you type. */
export function NewMessageDialog({ open, onOpenChange, onPick, myId }: NewMessageDialogProps) {
  const [query, setQuery] = useState('');
  const debounced = useDebounce(query.trim(), 300);
  const search = useSearchUsers(debounced, open, 10);
  const suggested = useSuggestedUsers(10);
  const source = debounced ? search : suggested;
  const users = (source.data?.items ?? []).filter((u) => u.id !== myId);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (!next) setQuery('');
      }}
    >
      <DialogContent className="gap-0 rounded-2xl p-0 sm:max-w-[480px]">
        <DialogTitle className="px-4 py-4 text-[20px] font-bold">New message</DialogTitle>
        <DialogDescription className="sr-only">Search for someone to message.</DialogDescription>
        <label className="flex items-center gap-3 border-y border-border px-4 py-3">
          <Search className="size-[18px] text-primary" />
          <input
            autoFocus
            placeholder="Search people"
            aria-label="Search people"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="min-w-0 flex-1 bg-transparent text-[15px] outline-none placeholder:text-muted-foreground"
          />
        </label>
        <div className="h-[360px] overflow-y-auto">
          {source.isPending ? (
            <p className="p-4 text-[15px] text-muted-foreground">Loading…</p>
          ) : users.length === 0 ? (
            <p className="p-4 text-[15px] text-muted-foreground">{debounced ? `No people found for "${debounced}"` : 'No suggestions yet.'}</p>
          ) : (
            users.map((user) => (
              <button
                key={user.id}
                type="button"
                onClick={() => onPick(user)}
                className="flex w-full cursor-pointer items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-foreground/[0.03]"
              >
                <UserAvatar user={user} />
                <span className="min-w-0">
                  <span className="block truncate text-[15px] font-bold">{displayNameOf(user)}</span>
                  <span className="block truncate text-[15px] text-muted-foreground">@{user.username}</span>
                </span>
              </button>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
