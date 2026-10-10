'use client'; // Error components must be Client Components

import Link from 'next/link';
import { useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { XLogo } from '@/components/XLogo';

export default function ErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Optionally log the error to an error reporting service like Sentry
    console.error('Root Error Boundary caught an error:', error);
  }, [error]);

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col items-center justify-center p-4">
      <div className="max-w-md text-center space-y-6">
        <div className="flex justify-center mb-8">
            <XLogo className="w-16 h-16 fill-foreground" />
        </div>
        
        <h1 className="text-3xl font-extrabold">Something went wrong</h1>
        
        <p className="text-muted-foreground text-[15px]">
          Don&apos;t worry, it&apos;s not you - it&apos;s us. We encountered an unexpected error while trying to load this page.
        </p>

        <div className="flex flex-col sm:flex-row gap-4 justify-center pt-4">
          <Button 
            onClick={() => reset()}
            className="rounded-full font-bold px-8 h-12"
          >
            Try again
          </Button>
          <Button asChild variant="outline" className="rounded-full font-bold px-8 h-12 border-border hover:bg-card">
            <Link href="/">Go to Home</Link>
          </Button>
        </div>
        
        {process.env.NODE_ENV === 'development' && (
            <div className="mt-8 p-4 bg-destructive/10 border border-destructive/20 rounded-xl text-left overflow-auto max-h-[300px]">
                <p className="font-mono text-xs text-destructive wrap-break-word">
                    {error.message || 'Unknown error'}
                </p>
                {error.stack && (
                     <pre className="font-mono text-[10px] text-destructive/80 mt-2 whitespace-pre-wrap">
                        {error.stack}
                     </pre>
                )}
            </div>
        )}
      </div>
    </div>
  );
}
