'use client';

import { useEffect, useState } from 'react';
import { useAuthStore } from '@/store/useAuthStore';
import { LoadingScreen } from '@/components/ui/LoadingScreen';
import { axiosInstance } from '@/api/axiosInstance';
import type { CurrentUserResponse } from '@/types';

export function AuthInitializer({ children }: { children: React.ReactNode }) {
  const setInitialized = useAuthStore((s) => s.setInitialized);
  const setAuth = useAuthStore((s) => s.setAuth);
  const logout = useAuthStore((s) => s.logout);
  const accessToken = useAuthStore((s) => s.accessToken);
  const refreshToken = useAuthStore((s) => s.refreshToken);
  
  // Local state to prevent hydration mismatch for isInitialized which starts false
  const [mounted, setMounted] = useState(false);
  const isInitialized = useAuthStore((s) => s.isInitialized);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setMounted(true);
  }, []);

  useEffect(() => {

    const init = async () => {
        // Prevent re-running if already initialized
        if (useAuthStore.getState().isInitialized) return;

        if (accessToken) {
            try {
                // Verify token and get fresh user data
                const { data } = await axiosInstance.get<CurrentUserResponse>('/auth/me');
                setAuth(accessToken, refreshToken!, data);
            } catch (error) {
                console.error('Auth initialization failed:', error);
                // If token is invalid (401), logout
                logout();
            }
        }
        setInitialized(true);
    };

    init();
  }, [accessToken, refreshToken, setAuth, logout, setInitialized]);

  if (!mounted || !isInitialized) {
    return <LoadingScreen />;
  }

  return <>{children}</>;
}
