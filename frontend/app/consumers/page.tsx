'use client';

import { useEffect, useState, useCallback } from 'react';
import { api, Consumer, Stream } from '@/lib/api';
import { useSSE, SSEEvent } from '@/lib/sse';
import { Server, Plus, Trash2, RefreshCw, Database } from 'lucide-react';
import Link from 'next/link';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { CardGridSkeleton, EmptyState, ErrorState } from '@/components/PageState';

export default function ConsumersPage() {
    const [streams, setStreams] = useState<Stream[]>([]);
    const [streamsLoading, setStreamsLoading] = useState(true);
    const [streamsError, setStreamsError] = useState<unknown>(null);
    const [selectedStream, setSelectedStream] = useState<string>('');

    const [consumers, setConsumers] = useState<Consumer[]>([]);
    const [consumersError, setConsumersError] = useState<unknown>(null);
    // Which stream `consumers` / `consumersError` belong to; anything else means still loading.
    const [loadedFor, setLoadedFor] = useState<string | null>(null);

    const [showCreateModal, setShowCreateModal] = useState(false);
    const toast = useToast();
    const confirm = useConfirm();

    const loadStreams = useCallback(async () => {
        setStreamsLoading(true);
        try {
            const data = await api.listStreams();
            setStreams(data);
            setStreamsError(null);
            // Keep the current selection if it still exists.
            setSelectedStream((current) =>
                current && data.some((s) => s.name === current) ? current : data[0]?.name ?? '',
            );
        } catch (err) {
            setStreamsError(err);
        } finally {
            setStreamsLoading(false);
        }
    }, []);

    const loadConsumers = useCallback(async () => {
        if (!selectedStream) return;

        try {
            const data = await api.listConsumers(selectedStream);
            setConsumers(data);
            setConsumersError(null);
        } catch (err) {
            setConsumersError(err);
        } finally {
            setLoadedFor(selectedStream);
        }
    }, [selectedStream]);

    useEffect(() => {
        loadStreams();
    }, [loadStreams]);

    useEffect(() => {
        loadConsumers();
    }, [loadConsumers]);

    const handleSSEEvent = useCallback((event: SSEEvent) => {
        if (event.type === 'consumer_created' || event.type === 'consumer_deleted') {
            loadConsumers();
        }
    }, [loadConsumers]);

    useSSE(handleSSEEvent);

    const handleDelete = async (name: string) => {
        const ok = await confirm({
            title: `Delete consumer "${name}"?`,
            message: `Applications reading from "${selectedStream}" through this consumer lose their position. Messages in the stream are kept.`,
        });
        if (!ok) return;

        try {
            await api.deleteConsumer(selectedStream, name);
            toast.success(`Consumer "${name}" deleted`);
            loadConsumers();
        } catch (err) {
            toast.error(`Failed to delete consumer "${name}"`, err);
        }
    };

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        Consumers
                    </h1>
                    <p className="text-muted-foreground">Manage stream consumers</p>
                </div>
                <div className="flex gap-3">
                    <select
                        value={selectedStream}
                        onChange={(e) => setSelectedStream(e.target.value)}
                        className="glass glass-hover rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    >
                        {streams.map((stream) => (
                            <option key={stream.name} value={stream.name}>
                                {stream.name}
                            </option>
                        ))}
                    </select>
                    <button
                        onClick={() => {
                            loadStreams();
                            loadConsumers();
                        }}
                        className="glass glass-hover rounded-lg px-4 py-2 flex items-center gap-2 text-foreground"
                    >
                        <RefreshCw className="w-4 h-4" />
                        Refresh
                    </button>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        disabled={!selectedStream}
                        className="bg-primary text-primary-foreground rounded-lg px-4 py-2 flex items-center gap-2 hover:bg-primary/90 transition-colors shadow-lg shadow-primary/20 disabled:opacity-50"
                    >
                        <Plus className="w-4 h-4" />
                        Create Consumer
                    </button>
                </div>
            </div>

            {streamsLoading && streams.length === 0 ? (
                <CardGridSkeleton />
            ) : streamsError ? (
                <ErrorState title="Couldn't load streams" error={streamsError} onRetry={loadStreams} />
            ) : streams.length === 0 ? (
                <EmptyState
                    icon={Database}
                    title="No streams yet"
                    description="Consumers read messages from a stream. Create a stream first, then come back to add consumers to it."
                >
                    <Link
                        href="/streams"
                        className="inline-block bg-primary text-primary-foreground rounded-lg px-6 py-3 hover:bg-primary/90 transition-colors"
                    >
                        Go to Streams
                    </Link>
                </EmptyState>
            ) : loadedFor !== selectedStream ? (
                <CardGridSkeleton count={2} />
            ) : consumersError ? (
                <ErrorState
                    title={`Couldn't load consumers for "${selectedStream}"`}
                    error={consumersError}
                    onRetry={loadConsumers}
                />
            ) : consumers.length === 0 ? (
                <EmptyState
                    icon={Server}
                    title={`No consumers on "${selectedStream}"`}
                    description="A durable consumer tracks which messages an application has processed. Create one to start reading from this stream."
                    actionLabel="Create a consumer"
                    onAction={() => setShowCreateModal(true)}
                />
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {consumers.map((consumer) => (
                        <div key={consumer.name} className="glass glass-hover rounded-xl p-6">
                            <div className="flex items-start justify-between mb-4">
                                <div>
                                    <h3 className="text-xl font-semibold mb-1 text-foreground">{consumer.name}</h3>
                                    <p className="text-sm text-muted-foreground">{consumer.stream_name}</p>
                                </div>
                                <button
                                    onClick={() => handleDelete(consumer.name)}
                                    className="text-destructive hover:text-destructive/80 transition-colors p-2 hover:bg-destructive/10 rounded-lg"
                                    aria-label={`Delete consumer ${consumer.name}`}
                                >
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </div>

                            <div className="grid grid-cols-3 gap-3 mb-4">
                                <div className="glass rounded-lg p-3">
                                    <p className="text-xl font-bold text-foreground">{consumer.num_pending}</p>
                                    <p className="text-xs text-muted-foreground">Pending</p>
                                </div>
                                <div className="glass rounded-lg p-3">
                                    <p className="text-xl font-bold text-foreground">{consumer.delivered}</p>
                                    <p className="text-xs text-muted-foreground">Delivered</p>
                                </div>
                                <div className="glass rounded-lg p-3">
                                    <p className="text-xl font-bold text-foreground">{consumer.num_redelivered}</p>
                                    <p className="text-xs text-muted-foreground">Redelivered</p>
                                </div>
                            </div>

                            <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">Deliver Policy:</span>
                                    <span className="text-foreground font-medium">{consumer.deliver_policy}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">Ack Policy:</span>
                                    <span className="text-foreground font-medium">{consumer.ack_policy}</span>
                                </div>
                                {consumer.filter_subject && (
                                    <div className="flex justify-between">
                                        <span className="text-muted-foreground">Filter:</span>
                                        <span className="text-foreground font-medium">{consumer.filter_subject}</span>
                                    </div>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {showCreateModal && selectedStream && (
                <CreateConsumerModal
                    streamName={selectedStream}
                    onClose={() => setShowCreateModal(false)}
                    onCreated={loadConsumers}
                />
            )}
        </div>
    );
}

function CreateConsumerModal({
    streamName,
    onClose,
    onCreated,
}: {
    streamName: string;
    onClose: () => void;
    onCreated: () => void;
}) {
    const [formData, setFormData] = useState({
        name: '',
        deliver_policy: 'all',
        ack_policy: 'explicit',
        filter_subject: '',
    });
    const [submitting, setSubmitting] = useState(false);
    const toast = useToast();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSubmitting(true);

        try {
            await api.createConsumer({
                stream_name: streamName,
                ...formData,
            });
            toast.success(`Consumer "${formData.name}" created`);
            onCreated();
            onClose();
        } catch (err) {
            toast.error('Failed to create consumer', err);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
            <div className="glass rounded-xl p-6 max-w-lg w-full">
                <h2 className="text-2xl font-bold mb-6 text-foreground">Create Consumer</h2>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Consumer Name</label>
                        <input
                            type="text"
                            required
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="my-consumer"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Stream</label>
                        <input
                            type="text"
                            disabled
                            value={streamName}
                            className="w-full bg-input/50 border border-border rounded-lg px-4 py-2 text-muted-foreground"
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium mb-2 text-foreground">
                                Deliver Policy
                            </label>
                            <select
                                value={formData.deliver_policy}
                                onChange={(e) => setFormData({ ...formData, deliver_policy: e.target.value })}
                                className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            >
                                <option value="all">All</option>
                                <option value="last">Last</option>
                                <option value="new">New</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium mb-2 text-foreground">Ack Policy</label>
                            <select
                                value={formData.ack_policy}
                                onChange={(e) => setFormData({ ...formData, ack_policy: e.target.value })}
                                className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            >
                                <option value="explicit">Explicit</option>
                                <option value="all">All</option>
                                <option value="none">None</option>
                            </select>
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">
                            Filter Subject (Optional)
                        </label>
                        <input
                            type="text"
                            value={formData.filter_subject}
                            onChange={(e) => setFormData({ ...formData, filter_subject: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="events.specific"
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
