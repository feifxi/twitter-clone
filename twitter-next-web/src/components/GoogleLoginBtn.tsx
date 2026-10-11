'use client';

import { GoogleLogin, CredentialResponse } from '@react-oauth/google';
import { useState, useRef, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { axiosInstance, getErrorMessage } from '@/api/axiosInstance';
import { queryKeys } from '@/lib/query-keys';
import type { CurrentUserResponse } from '@/types';
import type { GoogleAuthRequestInput } from '@/lib/validation';

import { useUIStore } from '@/store/useUIStore';

import { useQueryClient } from '@tanstack/react-query';

export default function GoogleLoginBtn() {
  const router = useRouter();
  const closeSignInModal = useUIStore((s) => s.closeSignInModal);
  const queryClient = useQueryClient();

  const [isLoading, setIsLoading] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const [buttonWidth, setButtonWidth] = useState<number>(364);

  useEffect(() => {
    const updateWidth = () => {
      if (containerRef.current) {
        setButtonWidth(containerRef.current.offsetWidth);
      }
    };
    
    updateWidth();
    const observer = new ResizeObserver(updateWidth);
    if (containerRef.current) observer.observe(containerRef.current);
    
    return () => observer.disconnect();
  }, []);

  const handleSuccess = async (credentialResponse: CredentialResponse) => {
    const googleToken = credentialResponse.credential;
    if (!googleToken) return;
    
    setIsLoading(true);
    try {
      const payload: GoogleAuthRequestInput = { idToken: googleToken };
      // The API sets the session cookies; the body is the signed-in user.
      const { data } = await axiosInstance.post<CurrentUserResponse>('/auth/google', payload);
      queryClient.setQueryData(queryKeys.auth.me, data);
      // Everything else was fetched anonymously (no likes, follows, …).
      await queryClient.invalidateQueries({ predicate: (q) => q.queryKey[0] !== 'auth' });

      closeSignInModal();
      router.push(data.onboarded ? '/' : '/onboarding');
      router.refresh(); // re-render server components with the new session
    } catch (error) {
      toast.error(getErrorMessage(error) || 'Sign-in failed. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div ref={containerRef} className="w-full h-[40px] relative overflow-hidden flex justify-center rounded-full">
      <GoogleLogin
        key={buttonWidth}
        onSuccess={handleSuccess}
        onError={() => toast.error('Google sign-in was cancelled or failed.')}
        theme="filled_black"
        shape="pill"
        size="large"
        width={buttonWidth.toString()}
        text="signup_with"
      />
      {isLoading && (
        <div className="absolute inset-0 bg-black/50 flex items-center justify-center rounded-full z-10 cursor-not-allowed">
           <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
        </div>
      )}
    </div>
  );
}
