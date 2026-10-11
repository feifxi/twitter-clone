'use client';

import { X } from 'lucide-react';
import { Dialog as DialogPrimitive } from 'radix-ui';

interface ImageLightboxProps {
  src: string | null;
  alt?: string;
  onClose: () => void;
}

/** Full-screen image viewer; Esc or a click outside the image closes it. */
export function ImageLightbox({ src, alt = 'Image', onClose }: ImageLightboxProps) {
  return (
    <DialogPrimitive.Root open={!!src} onOpenChange={(open) => !open && onClose()}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/90" />
        <DialogPrimitive.Content
          className="fixed inset-0 z-50 flex items-center justify-center p-4 outline-none"
          onClick={onClose}
        >
          <DialogPrimitive.Title className="sr-only">{alt}</DialogPrimitive.Title>
          <DialogPrimitive.Close
            className="absolute left-4 top-4 cursor-pointer rounded-full bg-black/50 p-2 text-white hover:bg-black/70"
            aria-label="Close image"
          >
            <X className="size-5" />
          </DialogPrimitive.Close>
          {src && (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={src} alt={alt} onClick={(e) => e.stopPropagation()} className="max-h-full max-w-full rounded-lg object-contain" />
          )}
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
