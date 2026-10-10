'use client';

import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';
import { cn } from '@/lib/utils';

interface ConfirmModalProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  danger?: boolean;
  pending?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}

/** X-style confirmation: big title, explanation, stacked pill buttons. */
export function ConfirmModal({ open, title, description, confirmLabel, danger, pending, onConfirm, onClose }: ConfirmModalProps) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && !pending && onClose()}>
      <DialogContent showCloseButton={false} className="max-w-[320px] rounded-2xl p-8" onClick={(e) => e.stopPropagation()}>
        <DialogTitle className="text-[20px] font-extrabold leading-6">{title}</DialogTitle>
        <DialogDescription className="text-[15px] leading-5">{description}</DialogDescription>
        <div className="mt-2 flex flex-col gap-3">
          <button
            type="button"
            onClick={onConfirm}
            disabled={pending}
            className={cn(
              'h-11 w-full cursor-pointer rounded-full text-[15px] font-bold transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60',
              danger ? 'bg-destructive text-white' : 'bg-foreground text-background',
            )}
          >
            {confirmLabel}
          </button>
          <button
            type="button"
            onClick={onClose}
            disabled={pending}
            className="h-11 w-full cursor-pointer rounded-full border border-border text-[15px] font-bold transition-colors hover:bg-accent disabled:opacity-60"
          >
            Cancel
          </button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
