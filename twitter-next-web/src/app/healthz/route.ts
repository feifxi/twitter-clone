// Liveness for the container health check (deploys wait for it). Publicly, Caddy routes
// /healthz to the API, so only the health check inside the container reaches this.
export const dynamic = 'force-dynamic';

export function GET() {
  return new Response('ok');
}
