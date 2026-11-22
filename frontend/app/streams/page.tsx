'use client';

import { useEffect, useState, useCallback } from 'react';
import { api, Stream } from '@/lib/api';
import { useSSE, SSEEvent } from '@/lib/sse';
import { Database, Plus, Trash2, RefreshCw } from 'lucide-react';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { CardGridSkeleton, EmptyState, ErrorState } from '@/components/PageState';

export default function StreamsPage() {
    const [streams, setStreams] = useState<Stream[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<unknown>(null);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const toast = useToast();
    const confirm = useConfirm();

    const loadStreams = useCallback(async () => {
        try {
            const data = await api.listStreams();
            setStreams(data);
            setError(null);
        } catch (err) {
            setError(err);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadStreams();
    }, [loadStreams]);

    const handleSSEEvent = useCallback((event: SSEEvent) => {
        if (event.type === 'stream_created' || event.type === 'stream_deleted') {
            loadStreams();
        }
    }, [loadStreams]);

    useSSE(handleSSEEvent);

    const handleDelete = async (name: string) => {
        const ok = await confirm({
            title: `Delete stream "${name}"?`,
            message: 'All messages in the stream and all of its consumers are deleted. This cannot be undone.',
        });
        if (!ok) return;

        try {
            await api.deleteStream(name);
            toast.success(`Stream "${name}" deleted`);
            loadStreams();
        } catch (err) {
            toast.error(`Failed to delete stream "${name}"`, err);
        }
    };

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        Streams
                    </h1>
                    <p className="text-muted-foreground">Manage your JetStream streams</p>
                </div>
                <div className="flex gap-3">
                    <button
                        onClick={loadStreams}
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
                        Create Stream
                    </button>
                </div>
            </div>

            {loading ? (
                <CardGridSkeleton />
            ) : error ? (
                <ErrorState title="Couldn't load streams" error={error} onRetry={loadStreams} />
            ) : streams.length === 0 ? (
                <EmptyState
                    icon={Database}
                    title="No streams yet"
                    description="A stream stores the messages published on one or more subjects. Create your first stream, then publish to it from the Messages page."
                    actionLabel="Create your first stream"
                    onAction={() => setShowCreateModal(true)}
                />
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {streams.map((stream) => (
                        <div key={stream.name} className="glass glass-hover rounded-xl p-6">
                            <div className="flex items-start justify-between mb-4">
                                <div>
                                    <h3 className="text-xl font-semibold mb-1 text-foreground">{stream.name}</h3>
                                    <p className="text-sm text-muted-foreground">
                                        {stream.subjects.join(', ')}
                                    </p>
                                </div>
                                <button
                                    onClick={() => handleDelete(stream.name)}
                                    className="text-destructive hover:text-destructive/80 transition-colors p-2 hover:bg-destructive/10 rounded-lg"
                                    aria-label={`Delete stream ${stream.name}`}
                                >
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </div>

                            <div className="grid grid-cols-2 gap-4 mb-4">
                                <div className="glass rounded-lg p-3">
                                    <p className="text-2xl font-bold text-foreground">{stream.messages.toLocaleString()}</p>
                                    <p className="text-xs text-muted-foreground">Messages</p>
                                </div>
                                <div className="glass rounded-lg p-3">
                                    <p className="text-2xl font-bold text-foreground">{formatBytes(stream.bytes)}</p>
                                    <p className="text-xs text-muted-foreground">Size</p>
                                </div>
                            </div>

                            <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">Storage:</span>
                                    <span className="text-foreground font-medium">{stream.storage}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">Retention:</span>
                                    <span className="text-foreground font-medium">{stream.retention}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">Replicas:</span>
                                    <span className="text-foreground font-medium">{stream.replicas}</span>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {showCreateModal && (
                <CreateStreamModal
                    onClose={() => setShowCreateModal(false)}
                    onCreated={loadStreams}
                />
            )}
        </div>
    );
}

function CreateStreamModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
    const [formData, setFormData] = useState({
        name: '',
        subjects: '',
        retention: 'limits',
        storage: 'file',
        max_msgs: 0,
        max_bytes: 0,
        max_age: 0,
    });
    const [submitting, setSubmitting] = useState(false);
    const toast = useToast();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSubmitting(true);

        try {
            await api.createStream({
                ...formData,
                subjects: formData.subjects.split(',').map(s => s.trim()).filter(Boolean),
            });
            toast.success(`Stream "${formData.name}" created`);
            onCreated();
            onClose();
        } catch (err) {
            toast.error('Failed to create stream', err);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
            <div className="glass rounded-xl p-6 max-w-lg w-full max-h-[90vh] overflow-y-auto">
                <h2 className="text-2xl font-bold mb-6 text-foreground">Create Stream</h2>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Stream Name</label>
                        <input
                            type="text"
                            required
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="my-stream"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">
                            Subjects (comma-separated)
                        </label>
                        <input
                            type="text"
                            required
                            value={formData.subjects}
                            onChange={(e) => setFormData({ ...formData, subjects: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="events.*, orders.>"
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium mb-2 text-foreground">Storage</label>
                            <select
                                value={formData.storage}
                                onChange={(e) => setFormData({ ...formData, storage: e.target.value })}
                                className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            >
                                <option value="file">File</option>
                                <option value="memory">Memory</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium mb-2 text-foreground">Retention</label>
                            <select
                                value={formData.retention}
                                onChange={(e) => setFormData({ ...formData, retention: e.target.value })}
                                className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            >
                                <option value="limits">Limits</option>
                                <option value="interest">Interest</option>
                                <option value="workqueue">Work Queue</option>
                            </select>
                        </div>
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
