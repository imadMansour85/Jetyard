const TOKEN_KEY = 'jetyard-token';

export interface AuthUser {
    username: string;
    role: string;
    created_at: string;
}

export function getToken(): string | null {
    if (typeof window === 'undefined') return null;
    try {
        return window.localStorage.getItem(TOKEN_KEY);
    } catch {
        return null;
    }
}

export function setToken(token: string) {
    try {
        window.localStorage.setItem(TOKEN_KEY, token);
    } catch {
        // Storage unavailable (private mode); the session lives for this page only.
    }
}

export function clearToken() {
    try {
        window.localStorage.removeItem(TOKEN_KEY);
    } catch {
        // nothing to do
    }
}

export function authHeaders(): Record<string, string> {
    const token = getToken();
    return token ? { Authorization: `Bearer ${token}` } : {};
}
