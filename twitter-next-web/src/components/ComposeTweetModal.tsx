'use client';

import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { CreateTweet } from './CreateTweet';
import { X } from 'lucide-react';

interface ComposeTweetModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function ComposeTweetModal({ isOpen, onClose }: ComposeTweetModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent showCloseButton={false} className="sm:max-w-[600px] bg-background border-border p-0 gap-0 top-[5%] translate-y-0 sm:top-[10%] min-h-[300px] flex flex-col">
        <DialogTitle className="sr-only">Compose new tweet</DialogTitle>
        <DialogDescription className="sr-only">Write and post a new tweet</DialogDescription>
        <div className="flex items-center h-[53px] px-4 shrink-0">
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="p-2 rounded-full hover:bg-accent transition-colors -ml-2 cursor-pointer"
          >
             <X size={20} className="text-foreground cursor-pointer" />
          </button>
        </div>
        
        <div className="pb-4">
            <CreateTweet autoFocus onSuccess={onClose} className="border-none px-4 py-2" />
        </div>
      </DialogContent>
    </Dialog>
  );
}
