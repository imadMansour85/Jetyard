'use client';

import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { AlertTriangle } from 'lucide-react';

export interface ConfirmOptions {
    title: string;
    message?: React.ReactNode;
    confirmLabel?: string;
    /** Red confirm button, for deletes. Defaults to true. */
    destructive?: boolean;
}

type ConfirmFn = (options: ConfirmOptions) => Promise<boolean>;

const ConfirmContext = createContext<ConfirmFn | null>(null);

interface PendingConfirm {
    options: ConfirmOptions;
    resolve: (confirmed: boolean) => void;
}

export function ConfirmProvider({ children }: { children: React.ReactNode }) {
    const [pending, setPending] = useState<PendingConfirm | null>(null);

    const confirm = useCallback<ConfirmFn>(
        (options) =>
            new Promise<boolean>((resolve) => {
                setPending({ options, resolve });
            }),
        [],
    );

    const close = useCallback(
        (confirmed: boolean) => {
            pending?.resolve(confirmed);
            setPending(null);
        },
        [pending],
    );

    useEffect(() => {
        if (!pending) return;
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') close(false);
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, [pending, close]);

    const destructive = pending?.options.destructive ?? true;

    return (
        <ConfirmContext.Provider value={confirm}>
            {children}
            {pending && (
                <div
                    className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-[55] animate-fade-in"
                    onClick={() => close(false)}
                >
                    <div
                        role="alertdialog"
                        aria-modal="true"
                        aria-labelledby="confirm-title"
                        className="glass rounded-xl p-6 max-w-md w-full"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <div className="flex items-start gap-3 mb-6">
                            {destructive && (
                                <div className="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center shrink-0">
                                    <AlertTriangle className="w-5 h-5 text-red-400" />
                                </div>
                            )}
                            <div>
                                <h2 id="confirm-title" className="text-lg font-semibold text-foreground mb-1">
                                    {pending.options.title}
                                </h2>
                                {pending.options.message && (
                                    <div className="text-sm text-muted-foreground">{pending.options.message}</div>
                                )}
                            </div>
                        </div>
                        <div className="flex gap-3">
                            {/* Cancel gets focus so Enter never confirms a delete by accident. */}
                            <button
                                autoFocus
                                onClick={() => close(false)}
                                className="flex-1 glass glass-hover rounded-lg px-4 py-2 text-foreground"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={() => close(true)}
                                className={`flex-1 rounded-lg px-4 py-2 transition-colors ${
                                    destructive
                                        ? 'bg-red-600 text-white hover:bg-red-500'
                                        : 'bg-primary text-primary-foreground hover:bg-primary/90'
                                }`}
                            >
                                {pending.options.confirmLabel ?? (destructive ? 'Delete' : 'Confirm')}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </ConfirmContext.Provider>
    );
}

/** Returns a function that opens the confirm dialog and resolves to the user's choice. */
export function useConfirm(): ConfirmFn {
    const ctx = useContext(ConfirmContext);
    if (!ctx) throw new Error('useConfirm must be used inside <ConfirmProvider>');
    return ctx;
}
