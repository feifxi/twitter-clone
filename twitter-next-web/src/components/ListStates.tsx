export interface EmptyCopy {
  title: string;
  body?: string;
}

/** X-style empty list: bold headline, muted explanation. */
export function EmptyState({ title, body }: EmptyCopy) {
  return (
    <div className="mx-auto max-w-[400px] px-8 py-12">
      <p className="text-[31px] font-extrabold leading-9">{title}</p>
      {body && <p className="mt-2 text-[15px] text-muted-foreground">{body}</p>}
    </div>
  );
}

/** A list that failed to load, with a retry button. */
export function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-4 px-8 py-10 text-center">
      <p className="text-[15px] text-muted-foreground">Something went wrong. Try reloading.</p>
      <button
        type="button"
        onClick={onRetry}
        className="cursor-pointer rounded-full bg-primary px-4 py-2 text-[15px] font-bold text-primary-foreground hover:bg-primary/90"
      >
        Retry
      </button>
    </div>
  );
}
