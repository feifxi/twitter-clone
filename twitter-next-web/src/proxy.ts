import { NextResponse, type NextRequest } from 'next/server';

const baseURL = process.env.NEXT_PUBLIC_API_URL;

/**
 * Renews an expired session before the page renders. The browser drops the
 * access cookie when it expires and keeps only the refresh cookie; without this,
 * server-rendered pages would show the signed-out state until the client
 * refreshed. New cookies go both to this render (request) and to the browser
 * (response).
 */
export async function proxy(request: NextRequest) {
  const refreshToken = request.cookies.get('refresh_token')?.value;
  if (!refreshToken || request.cookies.has('access_token')) {
    return NextResponse.next();
  }

  let refreshed: Response;
  try {
    refreshed = await fetch(`${baseURL}/auth/refresh`, {
      method: 'POST',
      headers: { cookie: `refresh_token=${refreshToken}` },
      cache: 'no-store',
    });
  } catch {
    return NextResponse.next(); // API unreachable: render anyway, the client retries
  }
  if (!refreshed.ok) {
    return NextResponse.next(); // the client sees TOKEN_EXPIRED and cleans up
  }

  const setCookies = refreshed.headers.getSetCookie();
  for (const header of setCookies) {
    const [pair] = header.split(';');
    const separator = pair.indexOf('=');
    request.cookies.set(pair.slice(0, separator).trim(), pair.slice(separator + 1).trim());
  }
  const response = NextResponse.next({ request: { headers: request.headers } });
  for (const header of setCookies) {
    response.headers.append('set-cookie', header);
  }
  return response;
}

export const config = {
  // Pages only: skip static assets and image optimization.
  matcher: ['/((?!_next/static|_next/image|favicon.ico|icon|.*\\.(?:png|jpg|jpeg|gif|svg|webp|ico)$).*)'],
};
