/** Error thrown by lib/api.ts for a non-2xx response. */
export class ApiError extends Error {
    readonly status: number;
    /** Backend request ID (X-Request-Id), to quote when reporting a problem. */
    readonly requestId?: string;

    constructor(message: string, status: number, requestId?: string) {
        super(message);
        this.name = 'ApiError';
        this.status = status;
        this.requestId = requestId;
    }
}

/** Human-readable text for anything caught in a catch block. */
export function errorMessage(err: unknown, fallback = 'Something went wrong'): string {
    if (err instanceof TypeError && err.message === 'Failed to fetch') {
        return 'Cannot reach the backend. Check that it is running and reachable.';
    }
    if (err instanceof ApiError) {
        // Server errors are the ones worth reporting, so give the ID for those.
        return err.requestId && err.status >= 500 ? `${err.message} (request ID: ${err.requestId})` : err.message;
    }
    if (err instanceof Error && err.message) return err.message;
    if (typeof err === 'string' && err) return err;
    return fallback;
}
