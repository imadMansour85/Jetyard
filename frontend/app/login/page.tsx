'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { Lock, Loader2 } from 'lucide-react';

export default function LoginPage() {
    const router = useRouter();
    const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [confirm, setConfirm] = useState('');
    const [error, setError] = useState('');
    const [submitting, setSubmitting] = useState(false);

    useEffect(() => {
        api.needsSetup()
            .then(setNeedsSetup)
            .catch(() => setError('Cannot reach the backend. Is it running?'));
    }, []);

    const submit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');

        if (needsSetup && password !== confirm) {
            setError('Passwords do not match');
            return;
        }

        setSubmitting(true);
        try {
            if (needsSetup) {
                await api.registerFirstAdmin(username, password);
            } else {
                await api.login(username, password);
            }
            router.push('/');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Login failed');
            setSubmitting(false);
        }
    };

    if (needsSetup === null && !error) {
        return (
            <div className="flex h-screen items-center justify-center bg-background">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
        );
    }

    return (
        <div className="flex h-screen items-center justify-center bg-background p-4">
            <div className="w-full max-w-sm">
                <div className="mb-8 text-center">
                    <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10">
                        <Lock className="h-6 w-6 text-primary" />
                    </div>
                    <h1 className="text-2xl font-semibold text-foreground">
                        {needsSetup ? 'Create admin account' : 'Jetyard'}
                    </h1>
                    <p className="mt-2 text-sm text-muted-foreground">
                        {needsSetup
                            ? 'This dashboard has no users yet. Create the first admin account.'
                            : 'Sign in to continue'}
                    </p>
                </div>

                <form onSubmit={submit} className="space-y-4 rounded-xl border border-border bg-card p-6">
                    <div>
                        <label htmlFor="username" className="text-sm font-medium text-foreground">
                            Username
                        </label>
                        <input
                            id="username"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            autoComplete="username"
                            required
                            className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
                        />
                    </div>

                    <div>
                        <label htmlFor="password" className="text-sm font-medium text-foreground">
                            Password
                        </label>
                        <input
                            id="password"
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            autoComplete={needsSetup ? 'new-password' : 'current-password'}
                            required
                            className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
                        />
                        {needsSetup && (
                            <p className="mt-1 text-xs text-muted-foreground">At least 12 characters.</p>
                        )}
                    </div>

                    {needsSetup && (
                        <div>
                            <label htmlFor="confirm" className="text-sm font-medium text-foreground">
                                Confirm password
                            </label>
                            <input
                                id="confirm"
                                type="password"
                                value={confirm}
                                onChange={(e) => setConfirm(e.target.value)}
                                autoComplete="new-password"
                                required
                                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
                            />
                        </div>
                    )}

                    {error && (
                        <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                            {error}
                        </div>
                    )}

                    <button
                        type="submit"
                        disabled={submitting}
                        className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:opacity-90 disabled:opacity-50"
                    >
                        {submitting && <Loader2 className="h-4 w-4 animate-spin" />}
                        {needsSetup ? 'Create account' : 'Sign in'}
                    </button>
                </form>
            </div>
        </div>
    );
}
