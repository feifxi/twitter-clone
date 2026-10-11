'use client';

import { Link2, MoreHorizontal, Trash2 } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

interface TweetActionsMenuProps {
  isOwn: boolean;
  onCopyLink: () => void;
  onDelete: () => void;
}

/** The "…" menu on a tweet. */
export function TweetActionsMenu({ isOwn, onCopyLink, onDelete }: TweetActionsMenuProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label="More options"
          onClick={(e) => e.stopPropagation()}
          className="-mr-2 cursor-pointer rounded-full p-2 text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary"
        >
          <MoreHorizontal className="size-[18px]" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56 rounded-xl py-1" onClick={(e) => e.stopPropagation()}>
        {isOwn && (
          <DropdownMenuItem variant="destructive" onSelect={onDelete} className="cursor-pointer gap-3 px-4 py-3 text-[15px] font-bold">
            <Trash2 className="size-[18px]" />
            Delete
          </DropdownMenuItem>
        )}
        <DropdownMenuItem onSelect={onCopyLink} className="cursor-pointer gap-3 px-4 py-3 text-[15px] font-bold">
          <Link2 className="size-[18px]" />
          Copy link
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
