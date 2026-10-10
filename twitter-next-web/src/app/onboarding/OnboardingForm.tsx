'use client';

import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { z } from 'zod';
import { axiosInstance, getErrorMessage } from '@/api/axiosInstance';
import { XLogo } from '@/components/XLogo';
import { queryKeys } from '@/lib/query-keys';
import type { CurrentUserResponse } from '@/types';

const schema = z.object({
  username: z
    .string()
    .trim()
    .min(3, 'At least 3 characters')
    .max(20, 'At most 20 characters')
    .regex(/^[A-Za-z0-9_]+$/, 'Only letters, numbers and underscores'),
  displayName: z.string().trim().min(1, "Name can't be blank").max(50, 'At most 50 characters'),
});
type OnboardingInput = z.infer<typeof schema>;

export function OnboardingForm({ user }: { user: CurrentUserResponse }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<OnboardingInput>({
    resolver: zodResolver(schema),
    defaultValues: { username: '', displayName: user.displayName ?? '' },
  });

  const submit = handleSubmit(async (data) => {
    try {
      const { data: me } = await axiosInstance.post<CurrentUserResponse>('/users/me/onboarding', data);
      queryClient.setQueryData(queryKeys.auth.me, me);
      router.replace('/');
      router.refresh();
    } catch (error) {
      const taken = isAxiosError(error) && error.response?.status === 409;
      setError('username', { message: taken ? 'That username is taken. Try another.' : getErrorMessage(error) });
    }
  });

  const field = 'h-14 w-full rounded-md border border-border bg-transparent px-3 text-[17px] outline-none focus:border-primary focus:ring-1 focus:ring-primary aria-[invalid=true]:border-destructive';

  return (
    <form onSubmit={submit} className="w-full max-w-[440px]" noValidate>
      <XLogo className="mx-auto size-10 text-foreground" />
      <h1 className="mt-8 text-[31px] font-extrabold leading-9">What should we call you?</h1>
      <p className="mt-2 text-[15px] text-muted-foreground">
        Your @username is unique. It&apos;s how people find and mention you.
      </p>

      <label className="mt-8 block">
        <span className="sr-only">Username</span>
        <div className="relative">
          <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[17px] text-muted-foreground">@</span>
          <input
            {...register('username')}
            autoFocus
            autoComplete="off"
            placeholder="username"
            aria-invalid={!!errors.username}
            className={`${field} pl-8`}
          />
        </div>
        {errors.username && <p className="mt-1 text-[13px] text-destructive">{errors.username.message}</p>}
      </label>

      <label className="mt-5 block">
        <span className="mb-1 block text-[13px] text-muted-foreground">Name</span>
        <input {...register('displayName')} aria-invalid={!!errors.displayName} className={field} />
        {errors.displayName && <p className="mt-1 text-[13px] text-destructive">{errors.displayName.message}</p>}
      </label>

      <button
        type="submit"
        disabled={isSubmitting}
        className="mt-10 h-[52px] w-full cursor-pointer rounded-full bg-foreground text-[17px] font-bold text-background hover:opacity-90 disabled:opacity-60"
      >
        {isSubmitting ? 'Saving…' : 'Next'}
      </button>
    </form>
  );
}
