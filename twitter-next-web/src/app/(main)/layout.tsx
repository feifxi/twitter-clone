import { redirect } from 'next/navigation';
import { getCurrentUser } from '@/api/server';
import { MainShell } from './MainShell';

export default async function MainLayout({ children }: { children: React.ReactNode }) {
  // New accounts pick their username before using the app (checked on the server: no flash).
  const user = await getCurrentUser();
  if (user && !user.onboarded) redirect('/onboarding');
  return <MainShell>{children}</MainShell>;
}
