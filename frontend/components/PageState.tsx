'use client';

import type { LucideIcon } from 'lucide-react';
import { AlertTriangle, RefreshCw } from 'lucide-react';
import { errorMessage } from '@/lib/errors';

/** Full-width error card for a page whose data failed to load. */
export function ErrorState({
    title,
    error,
    onRetry,
}: {
    title: string;
    error: unknown;
    onRetry: () => void;
}) {
    return (
        <div role="alert" className="glass rounded-xl p-12 text-center border-red-500/30">
            <AlertTriangle className="w-16 h-16 text-red-400 mx-auto mb-4" />
            <h3 className="text-xl font-semibold mb-2 text-foreground">{title}</h3>
            <p className="text-muted-foreground mb-6 break-words">{errorMessage(error)}</p>
            <button
                onClick={onRetry}
                className="bg-primary text-primary-foreground rounded-lg px-6 py-3 hover:bg-primary/90 transition-colors inline-flex items-center gap-2"
            >
                <RefreshCw className="w-4 h-4" />
                Retry
            </button>
        </div>
    );
}

/** Slim inline error banner, for a section of a page or a background refresh that failed. */
export function ErrorBanner({ message, error }: { message: string; error: unknown }) {
    return (
        <div role="alert" className="mb-4 flex items-center gap-2 p-3 rounded-lg text-sm bg-red-500/10 border border-red-500/30 text-red-400">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span className="break-words">
                {message}: {errorMessage(error)}
            </span>
        </div>
    );
}

/** Empty list placeholder that says what to do next. */
export function EmptyState({
    icon: Icon,
    title,
    description,
    actionLabel,
    onAction,
    children,
}: {
    icon: LucideIcon;
    title: string;
    description: React.ReactNode;
    actionLabel?: string;
    onAction?: () => void;
    children?: React.ReactNode;
}) {
    return (
        <div className="glass rounded-xl p-12 text-center">
            <Icon className="w-16 h-16 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-xl font-semibold mb-2 text-foreground">{title}</h3>
            <p className="text-muted-foreground mb-6 max-w-md mx-auto">{description}</p>
            {actionLabel && onAction && (
                <button
                    onClick={onAction}
                    className="bg-primary text-primary-foreground rounded-lg px-6 py-3 hover:bg-primary/90 transition-colors"
                >
                    {actionLabel}
                </button>
            )}
            {children}
        </div>
    );
}

function Bar({ className }: { className: string }) {
    return <div className={`rounded bg-muted/60 ${className}`} />;
}

/** Placeholder for the two-column card grids used by most list pages. */
export function CardGridSkeleton({ count = 4 }: { count?: number }) {
    return (
        <div aria-busy="true" aria-label="Loading" className="grid grid-cols-1 lg:grid-cols-2 gap-6 animate-pulse-soft">
            {Array.from({ length: count }, (_, i) => (
                <div key={i} className="glass rounded-xl p-6">
                    <Bar className="h-6 w-1/3 mb-2" />
                    <Bar className="h-4 w-1/2 mb-6" />
                    <div className="grid grid-cols-2 gap-4 mb-4">
                        <Bar className="h-16" />
                        <Bar className="h-16" />
                    </div>
                    <Bar className="h-4 w-full mb-2" />
                    <Bar className="h-4 w-5/6" />
                </div>
            ))}
        </div>
    );
}

/** Placeholder for single-column row lists (KV keys, messages). */
export function ListSkeleton({ rows = 5 }: { rows?: number }) {
    return (
        <div aria-busy="true" aria-label="Loading" className="space-y-3 animate-pulse-soft">
            {Array.from({ length: rows }, (_, i) => (
                <div key={i} className="glass rounded-lg p-4 flex items-center gap-3">
                    <Bar className="h-4 w-4" />
                    <Bar className="h-4 flex-1 max-w-xs" />
                </div>
            ))}
        </div>
    );
}
