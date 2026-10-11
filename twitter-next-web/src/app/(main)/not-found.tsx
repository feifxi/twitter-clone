import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="min-h-screen px-8 py-16 text-center">
      <h1 className="text-[31px] font-extrabold text-foreground">This page doesn&apos;t exist</h1>
      <p className="mt-2 text-[15px] text-muted-foreground">
        The account or post may have been deleted, or the link is wrong.
      </p>
      <Link
        href="/explore"
        className="mt-7 inline-block rounded-full bg-primary px-6 py-3 text-[15px] font-bold text-primary-foreground hover:bg-primary/90"
      >
        Search
      </Link>
    </div>
  );
}
