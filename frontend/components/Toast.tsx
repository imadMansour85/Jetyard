'use client';

import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react';
import { CheckCircle2, AlertTriangle, X } from 'lucide-react';
import { errorMessage } from '@/lib/errors';

type ToastKind = 'success' | 'error';

interface ToastItem {
    id: number;
    kind: ToastKind;
    message: string;
}

export interface ToastApi {
    success: (message: string) => void;
    /** Shows `message`, followed by the error's text when `err` is given. */
    error: (message: string, err?: unknown) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

const MAX_TOASTS = 4;
const DURATION_MS: Record<ToastKind, number> = { success: 4000, error: 8000 };

export function ToastProvider({ children }: { children: React.ReactNode }) {
    const [toasts, setToasts] = useState<ToastItem[]>([]);
    const nextId = useRef(0);

    const dismiss = useCallback((id: number) => {
        setToasts((current) => current.filter((t) => t.id !== id));
    }, []);

    const push = useCallback(
        (kind: ToastKind, message: string) => {
            const id = ++nextId.current;
            setToasts((current) => [...current.slice(-(MAX_TOASTS - 1)), { id, kind, message }]);
            setTimeout(() => dismiss(id), DURATION_MS[kind]);
        },
        [dismiss],
    );

    const api = useMemo<ToastApi>(
        () => ({
            success: (message) => push('success', message),
            error: (message, err) => push('error', err === undefined ? message : `${message}: ${errorMessage(err)}`),
        }),
        [push],
    );

    return (
        <ToastContext.Provider value={api}>
            {children}
            <div
                aria-live="polite"
                className="fixed bottom-4 right-4 z-[60] flex flex-col gap-2 w-[min(24rem,calc(100vw-2rem))] pointer-events-none"
            >
                {toasts.map((toast) => (
                    <div
                        key={toast.id}
                        role={toast.kind === 'error' ? 'alert' : 'status'}
                        className={`pointer-events-auto animate-slide-up flex items-start gap-3 rounded-lg border p-3 shadow-lg backdrop-blur-xl text-sm ${
                            toast.kind === 'error'
                                ? 'bg-red-950/90 border-red-500/40 text-red-100'
                                : 'bg-card/95 border-green-500/40 text-foreground'
                        }`}
                    >
                        {toast.kind === 'error' ? (
                            <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0 text-red-400" />
                        ) : (
                            <CheckCircle2 className="w-4 h-4 mt-0.5 shrink-0 text-green-400" />
                        )}
                        <p className="flex-1 break-words">{toast.message}</p>
                        <button
                            onClick={() => dismiss(toast.id)}
                            className="shrink-0 opacity-60 hover:opacity-100 transition-opacity"
                            aria-label="Dismiss notification"
                        >
                            <X className="w-4 h-4" />
                        </button>
                    </div>
                ))}
            </div>
        </ToastContext.Provider>
    );
}

export function useToast(): ToastApi {
    const ctx = useContext(ToastContext);
    if (!ctx) throw new Error('useToast must be used inside <ToastProvider>');
    return ctx;
}

/** Returns a function that copies text and confirms it with a toast. */
export function useCopyToClipboard(): (text: string) => Promise<void> {
    const toast = useToast();
    return useCallback(
        async (text: string) => {
            try {
                await navigator.clipboard.writeText(text);
                toast.success('Copied to clipboard');
            } catch {
                toast.error('Could not copy to clipboard');
            }
        },
        [toast],
    );
}
