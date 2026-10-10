import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { getCurrentUser } from '@/api/server';
import { OnboardingForm } from './OnboardingForm';

export const metadata: Metadata = { title: 'Pick a username / Chanom Twitter' };

/** First sign-in: new accounts get a generated handle; here they choose their own. */
export default async function OnboardingPage() {
  const user = await getCurrentUser();
  if (!user || user.onboarded) redirect('/');
  return (
    <main className="flex min-h-screen items-start justify-center bg-background px-4 pt-[10vh]">
      <OnboardingForm user={user} />
    </main>
  );
}
