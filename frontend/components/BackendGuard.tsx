'use client';

import { useEffect, useState } from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

import { API_BASE, apiLocation } from '@/lib/config';
const POLL_INTERVAL = 5000;

type Status = 'checking' | 'ok' | 'down';

export default function BackendGuard({ children }: { children: React.ReactNode }) {
    const [status, setStatus] = useState<Status>('checking');

    async function check() {
        try {
            const r = await fetch(`${API_BASE}/health`, { cache: 'no-store' });
            setStatus(r.ok ? 'ok' : 'down');
        } catch {
            setStatus('down');
        }
    }

    useEffect(() => {
        check();
        const id = setInterval(check, POLL_INTERVAL);
        return () => clearInterval(id);
    }, []);

    if (status === 'checking') {
        return (
            <div className="flex h-screen items-center justify-center bg-background">
                <div className="flex flex-col items-center gap-3 text-muted-foreground">
                    <div className="w-8 h-8 rounded-full border-2 border-primary border-t-transparent animate-spin" />
                    <p className="text-sm">Connecting to backend…</p>
                </div>
            </div>
        );
    }

    if (status === 'down') {
        return (
            <div className="flex h-screen items-center justify-center bg-background p-8">
                <div className="max-w-md w-full glass rounded-2xl border border-red-500/30 p-8 text-center space-y-5">
                    <div className="w-14 h-14 rounded-full bg-red-500/10 flex items-center justify-center mx-auto">
                        <AlertTriangle className="w-7 h-7 text-red-400" />
                    </div>

                    <div className="space-y-2">
                        <h1 className="text-xl font-bold text-foreground">Backend Unavailable</h1>
                        <p className="text-sm text-muted-foreground">
                            Can&apos;t reach the Jetyard backend at{' '}
                            <code className="text-xs bg-muted px-1 py-0.5 rounded break-all">{apiLocation()}</code>.
                        </p>
                    </div>

                    <div className="text-left bg-background rounded-lg border border-border p-4 space-y-2">
                        <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Common causes</p>
                        <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside">
                            <li>The backend isn&apos;t running, or exited at startup (check its logs)</li>
                            <li>
                                This site&apos;s origin isn&apos;t in the backend&apos;s{' '}
                                <code className="text-xs bg-muted px-1 py-0.5 rounded">ALLOWED_ORIGINS</code>, so the browser blocks the response
                            </li>
                            <li>
                                The backend can&apos;t reach NATS at{' '}
                                <code className="text-xs bg-muted px-1 py-0.5 rounded">NATS_URL</code>: it retries for about
                                a minute at startup, then exits
                            </li>
                        </ul>
                    </div>

                    <div className="flex gap-3 justify-center">
                        <button
                            onClick={() => { setStatus('checking'); check(); }}
                            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
                        >
                            <RefreshCw className="w-4 h-4" />
                            Retry
                        </button>
                    </div>

                    <p className="text-xs text-muted-foreground">
                        Retrying automatically every {POLL_INTERVAL / 1000} seconds…
                    </p>
                </div>
            </div>
        );
    }

    return <>{children}</>;
}
