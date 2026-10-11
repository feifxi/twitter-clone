'use client';

import { useEffect, useId, useRef, useState } from 'react';
import { useForm, useWatch, type UseFormRegisterReturn } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Camera, X } from 'lucide-react';
import { toast } from 'sonner';
import { getErrorMessage, getFieldErrors } from '@/api/axiosInstance';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';
import { UserAvatar } from '@/components/UserAvatar';
import { useUpdateProfile } from '@/hooks/useProfile';
import { cn } from '@/lib/utils';
import { updateProfileSchema, type UpdateProfileInput } from '@/lib/validation';
import type { UserResponse } from '@/types';

const IMAGE_TYPES = 'image/jpeg,image/png,image/gif,image/webp';
const MB = 1024 * 1024;

interface FloatingFieldProps {
  label: string;
  value: string;
  max: number;
  error?: string;
  multiline?: boolean;
  registration: UseFormRegisterReturn;
}

/** X-style input: the label sits inside and floats up on focus; the counter shows while focused. */
function FloatingField({ label, value, max, error, multiline, registration }: FloatingFieldProps) {
  const id = useId();
  const [focused, setFocused] = useState(false);
  const floated = focused || value.length > 0;
  const inputClass = 'w-full bg-transparent px-2 pb-2 pt-6 text-[17px] outline-none';
  const handlers = {
    ...registration,
    id,
    maxLength: max,
    onFocus: () => setFocused(true),
    onBlur: (e: React.FocusEvent<HTMLInputElement & HTMLTextAreaElement>) => {
      setFocused(false);
      registration.onBlur(e);
    },
    'aria-invalid': !!error,
    'aria-describedby': error ? `${id}-error` : undefined,
  };

  return (
    <div>
      <div
        className={cn(
          'relative rounded-md border transition-colors',
          error ? 'border-destructive' : focused ? 'border-primary ring-1 ring-primary' : 'border-border',
        )}
      >
        <label
          htmlFor={id}
          className={cn(
            'pointer-events-none absolute left-2 transition-all',
            floated ? 'top-1.5 text-[13px]' : 'top-4 text-[17px]',
            error ? 'text-destructive' : focused ? 'text-primary' : 'text-muted-foreground',
          )}
        >
          {label}
        </label>
        {focused && (
          <span className="absolute right-2 top-1.5 text-[13px] tabular-nums text-muted-foreground">
            {value.length} / {max}
          </span>
        )}
        {multiline ? <textarea rows={3} className={cn(inputClass, 'resize-none')} {...handlers} /> : <input className={inputClass} {...handlers} />}
      </div>
      {error && (
        <p id={`${id}-error`} className="mt-1 px-2 text-[13px] text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}

/** A picked image: the file plus a preview URL that is freed when replaced or unmounted. */
function useImagePick(maxBytes: number, what: string) {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [removed, setRemoved] = useState(false);
  useEffect(() => () => {
    if (preview) URL.revokeObjectURL(preview);
  }, [preview]);

  return {
    file,
    preview,
    removed,
    pick(e: React.ChangeEvent<HTMLInputElement>) {
      const next = e.target.files?.[0];
      e.target.value = '';
      if (!next) return;
      if (next.size > maxBytes) {
        toast.error(`That ${what} is too large (${maxBytes / MB} MB max).`);
        return;
      }
      setFile(next);
      setPreview(URL.createObjectURL(next));
      setRemoved(false);
    },
    remove() {
      setFile(null);
      setPreview(null);
      setRemoved(true);
    },
  };
}

function EditProfileForm({ user, onClose }: { user: UserResponse; onClose: () => void }) {
  const updateProfile = useUpdateProfile();
  const avatar = useImagePick(5 * MB, 'photo');
  const banner = useImagePick(10 * MB, 'header image');
  const avatarInput = useRef<HTMLInputElement>(null);
  const bannerInput = useRef<HTMLInputElement>(null);

  const {
    register,
    handleSubmit,
    control,
    setError,
    formState: { errors, isDirty },
  } = useForm<UpdateProfileInput>({
    resolver: zodResolver(updateProfileSchema),
    defaultValues: {
      displayName: user.displayName ?? '',
      bio: user.bio ?? '',
      location: user.location ?? '',
      website: user.website ?? '',
    },
    mode: 'onChange',
  });
  const values = useWatch({ control });

  const bannerSrc = banner.preview ?? (banner.removed ? null : user.bannerUrl);
  const avatarSrc = avatar.preview ?? (avatar.removed ? null : user.avatarUrl);
  const changed = isDirty || !!avatar.file || !!banner.file || avatar.removed || banner.removed;

  const save = handleSubmit(async (data) => {
    try {
      await updateProfile.mutateAsync({
        ...data,
        avatar: avatar.file ?? undefined,
        banner: banner.file ?? undefined,
        removeAvatar: avatar.removed,
        removeBanner: banner.removed,
      });
      toast.success('Profile saved.');
      onClose();
    } catch (error) {
      const fieldErrors = getFieldErrors(error);
      const fields = Object.keys(fieldErrors) as (keyof UpdateProfileInput)[];
      if (fields.length) fields.forEach((field) => setError(field, { message: fieldErrors[field] }));
      else toast.error(getErrorMessage(error));
    }
  });

  return (
    <form onSubmit={save} className="flex max-h-[90vh] flex-col">
      <div className="flex h-[53px] shrink-0 items-center justify-between px-4">
        <div className="flex items-center gap-8">
          <button type="button" onClick={onClose} aria-label="Close" className="-ml-2 cursor-pointer rounded-full p-2 hover:bg-accent">
            <X className="size-5" />
          </button>
          <DialogTitle className="text-[20px] font-bold">Edit profile</DialogTitle>
        </div>
        <button
          type="submit"
          disabled={!changed || updateProfile.isPending}
          className="h-8 cursor-pointer rounded-full bg-foreground px-4 text-[14px] font-bold text-background hover:opacity-90 disabled:cursor-default disabled:opacity-50"
        >
          {updateProfile.isPending ? 'Saving…' : 'Save'}
        </button>
      </div>
      <DialogDescription className="sr-only">Change your name, bio, location, website and pictures.</DialogDescription>

      <div className="overflow-y-auto pb-6">
        <div className="relative aspect-[3/1] bg-banner">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          {bannerSrc && <img src={bannerSrc} alt="" className="size-full object-cover" />}
          <div className="absolute inset-0 flex items-center justify-center gap-4 bg-black/30">
            <button type="button" onClick={() => bannerInput.current?.click()} aria-label="Change header image" className="cursor-pointer rounded-full bg-black/50 p-3 text-white hover:bg-black/70">
              <Camera className="size-5" />
            </button>
            {bannerSrc && (
              <button type="button" onClick={banner.remove} aria-label="Remove header image" className="cursor-pointer rounded-full bg-black/50 p-3 text-white hover:bg-black/70">
                <X className="size-5" />
              </button>
            )}
          </div>
          <input ref={bannerInput} type="file" accept={IMAGE_TYPES} className="hidden" onChange={banner.pick} />
        </div>

        <div className="relative -mt-12 ml-4 w-fit">
          <UserAvatar user={{ ...user, avatarUrl: avatarSrc }} size="xl" className="size-28 border-4 border-background" />
          <div className="absolute inset-1 flex items-center justify-center gap-2 rounded-full bg-black/45">
            <button type="button" onClick={() => avatarInput.current?.click()} aria-label="Change profile photo" className="cursor-pointer rounded-full bg-black/50 p-2.5 text-white hover:bg-black/70">
              <Camera className="size-5" />
            </button>
          </div>
          <input ref={avatarInput} type="file" accept={IMAGE_TYPES} className="hidden" onChange={avatar.pick} />
        </div>
        {avatarSrc && (
          <button type="button" onClick={avatar.remove} className="ml-4 mt-2 cursor-pointer text-[13px] text-muted-foreground hover:underline">
            Remove photo
          </button>
        )}

        <div className="mt-4 flex flex-col gap-6 px-4">
          <FloatingField label="Name" max={50} value={values.displayName ?? ''} error={errors.displayName?.message} registration={register('displayName')} />
          <FloatingField label="Bio" max={160} multiline value={values.bio ?? ''} error={errors.bio?.message} registration={register('bio')} />
          <FloatingField label="Location" max={30} value={values.location ?? ''} error={errors.location?.message} registration={register('location')} />
          <FloatingField label="Website" max={100} value={values.website ?? ''} error={errors.website?.message} registration={register('website')} />
        </div>
      </div>
    </form>
  );
}

interface EditProfileModalProps {
  user: UserResponse;
  isOpen: boolean;
  onClose: () => void;
}

export function EditProfileModal({ user, isOpen, onClose }: EditProfileModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent showCloseButton={false} className="top-[5%] translate-y-0 gap-0 overflow-hidden rounded-2xl p-0 sm:top-[8%] sm:max-w-[600px]">
        {/* Mounted only while open, so every open starts from the saved profile. */}
        {isOpen && <EditProfileForm user={user} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  );
}
