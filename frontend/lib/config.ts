// Where the backend API lives.
//
// Empty (the default) means same origin: relative URLs like /api/streams.
// That is how the single Docker image works, where the backend serves this
// frontend as static files. Set NEXT_PUBLIC_API_URL at build time when the
// API is on another origin, e.g. http://localhost:8080 for `next dev`
// (see .env.development) or a backend on another host.
export const API_BASE = (process.env.NEXT_PUBLIC_API_URL ?? '').replace(/\/+$/, '');

/** Human-readable API location, for error screens. */
export function apiLocation(): string {
    if (API_BASE) return API_BASE;
    return typeof window === 'undefined' ? '(same origin)' : window.location.origin;
}
