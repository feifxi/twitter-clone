"use client";

import { useEffect, useState } from "react";
import { GoogleOAuthProvider } from "@react-oauth/google";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider as NextThemesProvider } from "next-themes";
import { SignInModal } from "@/components/auth/SignInModal";
import { onSessionLost } from "@/api/axiosInstance";
import { queryKeys } from "@/lib/query-keys";
import { useUIStore } from "@/store/useUIStore";
import type { CurrentUserResponse } from "@/types";

type ProvidersProps = {
  children: React.ReactNode;
  /** The signed-in user from the server render; undefined when unknown. */
  initialUser: CurrentUserResponse | null | undefined;
};

export function Providers({ children, initialUser }: ProvidersProps) {
  // One client per browser tab (and per request on the server), never shared between users.
  const [queryClient] = useState(() => {
    const client = new QueryClient({
      // Data rendered on the server is fresh for a minute: don't refetch it right after hydration.
      defaultOptions: { queries: { staleTime: 60_000 } },
    });
    if (initialUser !== undefined) client.setQueryData(queryKeys.auth.me, initialUser);
    return client;
  });

  useEffect(
    () =>
      onSessionLost(({ request }) => {
        const wasSignedIn = !!queryClient.getQueryData(queryKeys.auth.me);
        queryClient.setQueryData(queryKeys.auth.me, null);
        // Viewer-specific data (likes, follows) was fetched as the old user.
        if (wasSignedIn) queryClient.invalidateQueries({ predicate: (q) => q.queryKey[0] !== "auth" });
        // An action (like, follow, post) needs a session: offer to sign in.
        if (request.method && request.method !== "get") useUIStore.getState().openSignInModal();
      }),
    [queryClient],
  );

  return (
    <NextThemesProvider attribute="class" defaultTheme="dark" enableSystem disableTransitionOnChange>
      <QueryClientProvider client={queryClient}>
        <GoogleOAuthProvider clientId={process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID!}>
          {children}
          <SignInModal />
        </GoogleOAuthProvider>
      </QueryClientProvider>
    </NextThemesProvider>
  );
}
