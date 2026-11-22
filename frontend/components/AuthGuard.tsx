'use client';

import { createContext, useContext, useEffect, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { clearToken, getToken, type AuthUser } from '@/lib/auth';

type State = 'checking' | 'authed' | 'anon';

export default function AuthGuard({ children }: { children: React.ReactNode }) {
    const router = useRouter();
    const pathname = usePathname();
    const [state, setState] = useState<State>('checking');
    const [user, setUser] = useState<AuthUser | null>(null);

    const isLoginPage = pathname === '/login';

    useEffect(() => {
        let cancelled = false;

        if (!getToken()) {
            setState('anon');
            if (!isLoginPage) router.replace('/login');
            return;
        }

        api.me()
            .then((u) => {
                if (cancelled) return;
                setUser(u);
                setState('authed');
                if (isLoginPage) router.replace('/');
            })
            .catch(() => {
                if (cancelled) return;
                clearToken();
                setState('anon');
                if (!isLoginPage) router.replace('/login');
            });

        return () => {
            cancelled = true;
        };
    }, [pathname, isLoginPage, router]);

    if (isLoginPage) return <>{children}</>;

    if (state !== 'authed') {
        return (
            <div className="flex h-screen items-center justify-center bg-background">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
            </div>
        );
    }

    return <AuthContext.Provider value={user}>{children}</AuthContext.Provider>;
}

const AuthContext = createContext<AuthUser | null>(null);

export function useCurrentUser() {
    return useContext(AuthContext);
}
