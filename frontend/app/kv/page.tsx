'use client';

import { useEffect, useState, useCallback } from 'react';
import { api, KVBucket } from '@/lib/api';
import { useSSE, SSEEvent } from '@/lib/sse';
import { Database, Plus, Trash2, RefreshCw, FolderOpen } from 'lucide-react';
import Link from 'next/link';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { CardGridSkeleton, EmptyState, ErrorState } from '@/components/PageState';

export default function KVPage() {
    const [buckets, setBuckets] = useState<KVBucket[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<unknown>(null);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const toast = useToast();
    const confirm = useConfirm();

    const loadBuckets = useCallback(async () => {
        try {
            const data = await api.listKVBuckets();
            setBuckets(data);
            setError(null);
        } catch (err) {
            setError(err);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadBuckets();
    }, [loadBuckets]);

    const handleSSEEvent = useCallback((event: SSEEvent) => {
        if (event.type === 'kv_bucket_created' || event.type === 'kv_bucket_deleted') {
            loadBuckets();
        }
    }, [loadBuckets]);

    useSSE(handleSSEEvent);

    const handleDelete = async (name: string) => {
        const ok = await confirm({
            title: `Delete KV bucket "${name}"?`,
            message: 'Every key and its history in this bucket are deleted. This cannot be undone.',
        });
        if (!ok) return;

        try {
            await api.deleteKVBucket(name);
            toast.success(`Bucket "${name}" deleted`);
            loadBuckets();
        } catch (err) {
            toast.error(`Failed to delete bucket "${name}"`, err);
        }
    };

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        KV Store
                    </h1>
                    <p className="text-muted-foreground">Manage JetStream key-value buckets</p>
                </div>
                <div className="flex gap-3">
                    <button
                        onClick={loadBuckets}
                        className="glass glass-hover rounded-lg px-4 py-2 flex items-center gap-2 text-foreground"
                    >
                        <RefreshCw className="w-4 h-4" />
                        Refresh
                    </button>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="bg-primary text-primary-foreground rounded-lg px-4 py-2 flex items-center gap-2 hover:bg-primary/90 transition-colors shadow-lg shadow-primary/20"
                    >
                        <Plus className="w-4 h-4" />
                        Create Bucket
                    </button>
                </div>
            </div>

            {loading ? (
                <CardGridSkeleton />
            ) : error ? (
                <ErrorState title="Couldn't load KV buckets" error={error} onRetry={loadBuckets} />
            ) : buckets.length === 0 ? (
                <EmptyState
                    icon={Database}
                    title="No KV buckets yet"
                    description="A bucket is a key-value store backed by JetStream, useful for config, feature flags or service state. Create one, then add keys to it."
                    actionLabel="Create your first bucket"
                    onAction={() => setShowCreateModal(true)}
                />
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {buckets.map((bucket) => (
                        <div key={bucket.name} className="glass glass-hover rounded-xl p-6">
                            <div className="flex items-start justify-between mb-4">
                                <div className="flex-1">
                                    <h3 className="text-xl font-semibold mb-1 text-foreground">{bucket.name}</h3>
                                    {bucket.description && (
                                        <p className="text-sm text-muted-foreground">{bucket.description}</p>
                                    )}
                                </div>
                                <button
                                    onClick={() => handleDelete(bucket.name)}
                                    className="text-destructive hover:text-destructive/80 transition-colors p-2 hover:bg-destructive/10 rounded-lg"
                                    aria-label={`Delete bucket ${bucket.name}`}
                                >
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </div>

                            <div className="grid grid-cols-2 gap-4 mb-4">
                                <div className="glass rounded-lg p-3">
                                    <p className="text-2xl font-bold text-foreground">{bucket.values.toLocaleString()}</p>
                                    <p className="text-xs text-muted-foreground">Keys</p>
                                </div>
                                <div className="glass rounded-lg p-3">
                                    <p className="text-2xl font-bold text-foreground">{formatBytes(bucket.bytes)}</p>
                                    <p className="text-xs text-muted-foreground">Size</p>
                                </div>
                            </div>

                            <div className="space-y-2 text-sm mb-4">
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">History:</span>
                                    <span className="text-foreground font-medium">{bucket.history} revisions</span>
                                </div>
                                {bucket.ttl > 0 && (
                                    <div className="flex justify-between">
                                        <span className="text-muted-foreground">TTL:</span>
                                        <span className="text-foreground font-medium">{formatDuration(bucket.ttl)}</span>
                                    </div>
                                )}
                            </div>

                            <Link
                                href={`/kv/bucket?name=${encodeURIComponent(bucket.name)}`}
                                className="w-full bg-primary text-primary-foreground rounded-lg px-4 py-2 hover:bg-primary/90 transition-colors flex items-center justify-center gap-2"
                            >
                                <FolderOpen className="w-4 h-4" />
                                View Keys
                            </Link>
                        </div>
                    ))}
                </div>
            )}

            {showCreateModal && (
                <CreateBucketModal
                    onClose={() => setShowCreateModal(false)}
                    onCreated={loadBuckets}
                />
            )}
        </div>
    );
}

function CreateBucketModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
    const [formData, setFormData] = useState({
        name: '',
        description: '',
        history: 1,
        ttl: 0,
    });
    const [submitting, setSubmitting] = useState(false);
    const toast = useToast();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSubmitting(true);

        try {
            await api.createKVBucket({
                name: formData.name,
                description: formData.description || undefined,
                history: formData.history,
                ttl: formData.ttl > 0 ? formData.ttl * 1000000000 : 0, // Convert seconds to nanoseconds
            });
            toast.success(`Bucket "${formData.name}" created`);
            onCreated();
            onClose();
        } catch (err) {
            toast.error('Failed to create bucket', err);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
            <div className="glass rounded-xl p-6 max-w-lg w-full max-h-[90vh] overflow-y-auto">
                <h2 className="text-2xl font-bold mb-6 text-foreground">Create KV Bucket</h2>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Bucket Name</label>
                        <input
                            type="text"
                            required
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="my-bucket"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">
                            Description (optional)
                        </label>
                        <input
                            type="text"
                            value={formData.description}
                            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="My KV bucket for..."
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">History (revisions)</label>
                        <input
                            type="number"
                            min="1"
                            max="64"
                            value={formData.history}
                            onChange={(e) => setFormData({ ...formData, history: parseInt(e.target.value) })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        />
                        <p className="text-xs text-muted-foreground mt-1">Number of historical values to keep per key</p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">TTL (seconds, 0 = no expiry)</label>
                        <input
                            type="number"
                            min="0"
                            value={formData.ttl}
                            onChange={(e) => setFormData({ ...formData, ttl: parseInt(e.target.value) })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
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
                            {submitting ? 'Creating...' : 'Create'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}

function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}

function formatDuration(nanoseconds: number): string {
    const seconds = nanoseconds / 1000000000;
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
    return `${Math.floor(seconds / 86400)}d`;
}
