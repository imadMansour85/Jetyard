'use client';

import { Suspense, useEffect, useState, useCallback } from 'react';
import { api, KVEntry } from '@/lib/api';
import { useSearchParams } from 'next/navigation';
import { ArrowLeft, Plus, Trash2, RefreshCw, Key, Copy, Edit } from 'lucide-react';
import Link from 'next/link';
import { useCopyToClipboard, useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { EmptyState, ErrorState, ListSkeleton } from '@/components/PageState';
import { errorMessage } from '@/lib/errors';

// /kv/bucket?name=<bucket>. A query parameter rather than /kv/[bucket] so the
// page can be exported statically and served by the backend.
export default function BucketKeysPage() {
    // useSearchParams needs a Suspense boundary in a statically exported page.
    return (
        <Suspense fallback={<ListSkeleton />}>
            <BucketKeys />
        </Suspense>
    );
}

function BucketKeys() {
    const bucket = useSearchParams().get('name') ?? '';

    const [keys, setKeys] = useState<string[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<unknown>(null);
    const [showPutModal, setShowPutModal] = useState(false);
    const [editingKey, setEditingKey] = useState<string | null>(null);
    const toast = useToast();
    const confirm = useConfirm();

    const loadKeys = useCallback(async () => {
        if (!bucket) {
            setError('No bucket selected. Open a bucket from the KV Store page.');
            setLoading(false);
            return;
        }
        try {
            // The backend returns [] for an empty bucket, so any error here is real.
            const data = await api.listKeys(bucket);
            setKeys(data || []);
            setError(null);
        } catch (err) {
            setError(err);
        } finally {
            setLoading(false);
        }
    }, [bucket]);

    useEffect(() => {
        loadKeys();
    }, [loadKeys]);

    const handleDelete = async (key: string) => {
        const ok = await confirm({
            title: `Delete key "${key}"?`,
            message: `The key is removed from "${bucket}". Earlier revisions stay in the bucket history until they age out.`,
        });
        if (!ok) return;

        try {
            await api.deleteKey(bucket, key);
            toast.success(`Key "${key}" deleted`);
            loadKeys();
        } catch (err) {
            toast.error(`Failed to delete key "${key}"`, err);
        }
    };

    const copyToClipboard = useCopyToClipboard();

    return (
        <div className="animate-fade-in">
            <div className="mb-6">
                <Link
                    href="/kv"
                    className="inline-flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
                >
                    <ArrowLeft className="w-4 h-4" />
                    Back to buckets
                </Link>
            </div>

            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        {bucket}
                    </h1>
                    <p className="text-muted-foreground">Manage keys in this bucket</p>
                </div>
                <div className="flex gap-3">
                    <button
                        onClick={loadKeys}
                        className="glass glass-hover rounded-lg px-4 py-2 flex items-center gap-2 text-foreground"
                    >
                        <RefreshCw className="w-4 h-4" />
                        Refresh
                    </button>
                    <button
                        onClick={() => {
                            setEditingKey(null);
                            setShowPutModal(true);
                        }}
                        className="bg-primary text-primary-foreground rounded-lg px-4 py-2 flex items-center gap-2 hover:bg-primary/90 transition-colors shadow-lg shadow-primary/20"
                    >
                        <Plus className="w-4 h-4" />
                        Add Key
                    </button>
                </div>
            </div>

            {loading ? (
                <ListSkeleton />
            ) : error ? (
                <ErrorState title={`Couldn't load keys in "${bucket}"`} error={error} onRetry={loadKeys} />
            ) : keys.length === 0 ? (
                <EmptyState
                    icon={Key}
                    title="This bucket is empty"
                    description="Add a key-value pair. Values can be any text, such as JSON config."
                    actionLabel="Add your first key"
                    onAction={() => {
                        setEditingKey(null);
                        setShowPutModal(true);
                    }}
                />
            ) : (
                <div className="space-y-3">
                    {keys.map((key) => (
                        <KeyRow
                            key={key}
                            bucketName={bucket}
                            keyName={key}
                            onEdit={() => {
                                setEditingKey(key);
                                setShowPutModal(true);
                            }}
                            onDelete={() => handleDelete(key)}
                            onCopy={() => copyToClipboard(key)}
                        />
                    ))}
                </div>
            )}

            {showPutModal && (
                <PutKeyModal
                    bucket={bucket}
                    initialKey={editingKey}
                    onClose={() => {
                        setShowPutModal(false);
                        setEditingKey(null);
                    }}
                    onSaved={loadKeys}
                />
            )}
        </div>
    );
}

function KeyRow({
    bucketName,
    keyName,
    onEdit,
    onDelete,
    onCopy,
}: {
    bucketName: string;
    keyName: string;
    onEdit: () => void;
    onDelete: () => void;
    onCopy: () => void;
}) {
    const [entry, setEntry] = useState<KVEntry | null>(null);
    const [expanded, setExpanded] = useState(false);
    const [loading, setLoading] = useState(false);
    const toast = useToast();

    const loadValue = async () => {
        setLoading(true);
        try {
            const data = await api.getKey(bucketName, keyName);
            setEntry(data);
            setExpanded(true);
        } catch (err) {
            toast.error(`Failed to load "${keyName}"`, err);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="glass glass-hover rounded-lg p-4">
            <div className="flex items-center justify-between">
                <button
                    onClick={() => (expanded ? setExpanded(false) : loadValue())}
                    className="flex-1 flex items-center gap-3 text-left"
                    disabled={loading}
                >
                    <Key className="w-4 h-4 text-primary" />
                    <span className="font-mono text-foreground">{keyName}</span>
                    {entry && (
                        <span className="text-xs text-muted-foreground">rev: {entry.revision}</span>
                    )}
                </button>
                <div className="flex items-center gap-2">
                    <button
                        onClick={onCopy}
                        className="text-muted-foreground hover:text-foreground transition-colors p-2 rounded-lg hover:bg-card/50"
                        title="Copy key"
                    >
                        <Copy className="w-4 h-4" />
                    </button>
                    <button
                        onClick={onEdit}
                        className="text-primary hover:text-primary/80 transition-colors p-2 rounded-lg hover:bg-primary/10"
                        title="Edit value"
                    >
                        <Edit className="w-4 h-4" />
                    </button>
                    <button
                        onClick={onDelete}
                        className="text-destructive hover:text-destructive/80 transition-colors p-2 rounded-lg hover:bg-destructive/10"
                        title="Delete key"
                    >
                        <Trash2 className="w-4 h-4" />
                    </button>
                </div>
            </div>

            {expanded && entry && (
                <div className="mt-4 pt-4 border-t border-border/50">
                    <div className="glass rounded-lg p-3 mb-3">
                        <div className="text-xs text-muted-foreground mb-2">Value:</div>
                        <pre className="text-sm text-foreground font-mono whitespace-pre-wrap break-all">
                            {entry.value}
                        </pre>
                    </div>
                    <div className="flex gap-4 text-xs text-muted-foreground">
                        <span>Created: {new Date(entry.created).toLocaleString()}</span>
                        <span>Operation: {entry.operation}</span>
                    </div>
                </div>
            )}
        </div>
    );
}

function PutKeyModal({
    bucket,
    initialKey,
    onClose,
    onSaved,
}: {
    bucket: string;
    initialKey: string | null;
    onClose: () => void;
    onSaved: () => void;
}) {
    const [key, setKey] = useState(initialKey || '');
    const [value, setValue] = useState('');
    const [loadError, setLoadError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const toast = useToast();

    useEffect(() => {
        if (initialKey) {
            api.getKey(bucket, initialKey)
                .then((entry) => setValue(entry.value))
                .catch((err) => setLoadError(errorMessage(err)));
        }
    }, [bucket, initialKey]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSubmitting(true);

        try {
            await api.putKey(bucket, key, value);
            toast.success(`Key "${key}" saved`);
            onSaved();
            onClose();
        } catch (err) {
            toast.error(`Failed to save key "${key}"`, err);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
            <div className="glass rounded-xl p-6 max-w-2xl w-full">
                <h2 className="text-2xl font-bold mb-6 text-foreground">
                    {initialKey ? 'Edit Key' : 'Add Key'}
                </h2>

                <form onSubmit={handleSubmit} className="space-y-4">
                    {loadError && (
                        <p role="alert" className="text-sm text-red-400">
                            Couldn&apos;t load the current value: {loadError}. Saving replaces it.
                        </p>
                    )}
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Key</label>
                        <input
                            type="text"
                            required
                            value={key}
                            onChange={(e) => setKey(e.target.value)}
                            disabled={!!initialKey}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
                            placeholder="my-key"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Value</label>
                        <textarea
                            required
                            value={value}
                            onChange={(e) => setValue(e.target.value)}
                            rows={6}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary font-mono text-sm"
                            placeholder='{"example": "value"}'
                        />
                    </div>

                    <div className="flex gap-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="flex-1 glass glass-hover rounded-lg px-4 py-2 text-foreground"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={submitting}
                            className="flex-1 bg-primary text-primary-foreground rounded-lg px-4 py-2 hover:bg-primary/90 transition-colors disabled:opacity-50"
                        >
                            {submitting ? 'Saving...' : 'Save'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
