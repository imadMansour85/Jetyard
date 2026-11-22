'use client';

import { useState, useCallback, useEffect } from 'react';
import { api, Stream, Message } from '@/lib/api';
import { MessageSquare, Send, RefreshCw, Database } from 'lucide-react';
import { useToast } from '@/components/Toast';
import { ErrorBanner, ErrorState, ListSkeleton } from '@/components/PageState';

export default function MessagesPage() {
    const [streams, setStreams] = useState<Stream[]>([]);
    const [streamsError, setStreamsError] = useState<unknown>(null);
    const [selectedStream, setSelectedStream] = useState<string>('');
    const [messages, setMessages] = useState<Message[]>([]);
    const [messagesError, setMessagesError] = useState<unknown>(null);
    const [loading, setLoading] = useState(false);
    const toast = useToast();

    const [publishData, setPublishData] = useState({
        subject: '',
        data: '',
    });
    const [publishing, setPublishing] = useState(false);

    const loadStreams = useCallback(async () => {
        try {
            const data = await api.listStreams();
            setStreams(data);
            setStreamsError(null);
        } catch (err) {
            setStreamsError(err);
        }
    }, []);

    const loadMessages = useCallback(async () => {
        if (!selectedStream) return;

        setLoading(true);
        try {
            const data = await api.getMessages(selectedStream, 20);
            setMessages(data);
            setMessagesError(null);
        } catch (err) {
            setMessagesError(err);
        } finally {
            setLoading(false);
        }
    }, [selectedStream]);

    useEffect(() => {
        loadStreams();
    }, [loadStreams]);

    useEffect(() => {
        if (selectedStream) {
            loadMessages();
        }
    }, [selectedStream, loadMessages]);

    const handlePublish = async (e: React.FormEvent) => {
        e.preventDefault();
        setPublishing(true);

        try {
            await api.publishMessage({
                subject: publishData.subject,
                data: publishData.data,
            });

            setPublishData({ subject: '', data: '' });
            toast.success(`Published to "${publishData.subject}"`);

            // Refresh the message list; the stream may have captured it.
            if (selectedStream) {
                setTimeout(loadMessages, 500);
            }
        } catch (err) {
            toast.error('Failed to publish message', err);
        } finally {
            setPublishing(false);
        }
    };

    return (
        <div className="animate-fade-in">
            <div className="mb-8">
                <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                    Messages
                </h1>
                <p className="text-muted-foreground">Publish and view messages</p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                {/* Publish Message */}
                <div className="glass rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-6">
                        <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-green-500 to-green-600 flex items-center justify-center">
                            <Send className="w-5 h-5 text-white" />
                        </div>
                        <h2 className="text-xl font-semibold text-foreground">Publish Message</h2>
                    </div>

                    <form onSubmit={handlePublish} className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium mb-2 text-foreground">Subject</label>
                            <input
                                type="text"
                                required
                                value={publishData.subject}
                                onChange={(e) => setPublishData({ ...publishData, subject: e.target.value })}
                                className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                placeholder="events.user.signup"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium mb-2 text-foreground">
                                Message Data
                            </label>
                            <textarea
                                required
                                value={publishData.data}
                                onChange={(e) => setPublishData({ ...publishData, data: e.target.value })}
                                rows={6}
                                className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary font-mono text-sm"
                                placeholder='{"user": "john", "action": "signup"}'
                            />
                        </div>

                        <button
                            type="submit"
                            disabled={publishing}
                            className="w-full bg-primary text-primary-foreground rounded-lg px-4 py-3 hover:bg-primary/90 transition-colors shadow-lg shadow-primary/20 disabled:opacity-50 flex items-center justify-center gap-2"
                        >
                            <Send className="w-4 h-4" />
                            {publishing ? 'Publishing...' : 'Publish Message'}
                        </button>
                    </form>
                </div>

                {/* Stream Info */}
                <div className="glass rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-6">
                        <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center">
                            <Database className="w-5 h-5 text-white" />
                        </div>
                        <h2 className="text-xl font-semibold text-foreground">Stream Messages</h2>
                    </div>

                    {streamsError ? <ErrorBanner message="Couldn't load streams" error={streamsError} /> : null}
                    {!streamsError && streams.length === 0 && (
                        <p className="text-sm text-muted-foreground mb-4">
                            No streams yet. Messages published to a subject that no stream captures are delivered to
                            live subscribers only and not stored.
                        </p>
                    )}
                    <div className="mb-4">
                        <label className="block text-sm font-medium mb-2 text-foreground">Select Stream</label>
                        <div className="flex gap-2">
                            <select
                                value={selectedStream}
                                onChange={(e) => {
                                    setMessages([]);
                                    setMessagesError(null);
                                    setSelectedStream(e.target.value);
                                }}
                                className="flex-1 bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            >
                                <option value="">Select a stream...</option>
                                {streams.map((stream) => (
                                    <option key={stream.name} value={stream.name}>
                                        {stream.name}
                                    </option>
                                ))}
                            </select>
                            <button
                                onClick={() => {
                                    loadStreams();
                                    loadMessages();
                                }}
                                aria-label="Refresh streams and messages"
                                className="glass glass-hover rounded-lg px-4 py-2"
                            >
                                <RefreshCw className="w-4 h-4" />
                            </button>
                        </div>
                    </div>

                    {selectedStream && (
                        <div className="glass rounded-lg p-4">
                            <p className="text-sm text-muted-foreground mb-2">Stream subjects:</p>
                            <p className="text-sm text-foreground font-mono">
                                {streams.find(s => s.name === selectedStream)?.subjects.join(', ')}
                            </p>
                        </div>
                    )}
                </div>
            </div>

            {/* Message List */}
            {selectedStream && (
                <div className="glass rounded-xl p-6">
                    <div className="flex items-center justify-between mb-6">
                        <h2 className="text-xl font-semibold text-foreground">
                            Recent Messages from {selectedStream}
                        </h2>
                        <span className="text-sm text-muted-foreground">
                            {messages.length} message{messages.length !== 1 ? 's' : ''}
                        </span>
                    </div>

                    {loading && messages.length === 0 ? (
                        <ListSkeleton rows={3} />
                    ) : messagesError ? (
                        <ErrorState
                            title={`Couldn't load messages from "${selectedStream}"`}
                            error={messagesError}
                            onRetry={loadMessages}
                        />
                    ) : messages.length === 0 ? (
                        <div className="text-center py-12">
                            <MessageSquare className="w-16 h-16 text-muted-foreground mx-auto mb-4" />
                            <p className="text-muted-foreground">
                                This stream is empty. Publish a message on one of its subjects using the form above.
                            </p>
                        </div>
                    ) : (
                        <div className="space-y-3">
                            {messages.map((message) => (
                                <div key={message.sequence} className="glass glass-hover rounded-lg p-4">
                                    <div className="flex items-start justify-between mb-2">
                                        <div className="flex items-center gap-3">
                                            <span className="text-xs font-mono bg-primary/20 text-primary px-2 py-1 rounded">
                                                #{message.sequence}
                                            </span>
                                            <span className="text-sm font-medium text-foreground">{message.subject}</span>
                                        </div>
                                        <span className="text-xs text-muted-foreground">
                                            {new Date(message.time).toLocaleString()}
                                        </span>
                                    </div>
                                    <pre className="text-sm bg-background/50 rounded p-3 overflow-x-auto text-foreground font-mono">
                                        {message.data}
                                    </pre>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
