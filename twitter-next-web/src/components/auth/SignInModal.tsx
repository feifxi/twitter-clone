'use client';

import { X } from 'lucide-react';
import GoogleLoginBtn from '@/components/GoogleLoginBtn';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';
import { XLogo } from '@/components/XLogo';
import { useUIStore } from '@/store/useUIStore';

/** Google is the only sign-in method; the first sign-in creates the account. */
export function SignInModal() {
  const isOpen = useUIStore((s) => s.isSignInModalOpen);
  const close = useUIStore((s) => s.closeSignInModal);

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && close()}>
      <DialogContent showCloseButton={false} className="gap-0 overflow-hidden rounded-2xl p-0 sm:max-w-[600px]">
        <div className="relative flex h-[53px] items-center px-4">
          <button type="button" onClick={close} aria-label="Close" className="-ml-2 cursor-pointer rounded-full p-2 transition-colors hover:bg-accent">
            <X className="size-5" />
          </button>
          <XLogo className="absolute left-1/2 size-8 -translate-x-1/2 text-foreground" />
        </div>

        <div className="mx-auto w-full max-w-[364px] px-8 pb-16 pt-8 sm:px-0">
          <DialogTitle className="text-[31px] font-extrabold leading-9">Sign in to Chanom Twitter</DialogTitle>
          <DialogDescription className="mt-3 text-[15px] text-muted-foreground">
            Use your Google account. New here? Signing in creates your account, and you&apos;ll pick a username next.
          </DialogDescription>
          <div className="mt-8">
            <GoogleLoginBtn />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
