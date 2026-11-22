'use client';

import { useEffect, useState } from 'react';
import { api, Stream, MessageDetail } from '@/lib/api';
import { Search, RefreshCw, Eye, Copy, ChevronDown, ChevronUp, Database } from 'lucide-react';
import Link from 'next/link';
import { useCopyToClipboard } from '@/components/Toast';
import { EmptyState, ErrorState, ListSkeleton } from '@/components/PageState';

export default function MessageBrowserPage() {
    const [streams, setStreams] = useState<Stream[]>([]);
    const [streamsLoading, setStreamsLoading] = useState(true);
    const [streamsError, setStreamsError] = useState<unknown>(null);
    const [selectedStream, setSelectedStream] = useState<string>('');
    const [messages, setMessages] = useState<MessageDetail[]>([]);
    const [messagesError, setMessagesError] = useState<unknown>(null);
    const [loading, setLoading] = useState(false);
    const [subjectFilter, setSubjectFilter] = useState('');
    const [keywordFilter, setKeywordFilter] = useState('');

    useEffect(() => {
        loadStreams();
    }, []);

    const loadStreams = async () => {
        setStreamsLoading(true);
        try {
            const data = await api.listStreams();
            setStreams(data);
            setStreamsError(null);
            setSelectedStream((current) =>
                current && data.some((s) => s.name === current) ? current : data[0]?.name ?? '',
            );
        } catch (err) {
            setStreamsError(err);
        } finally {
            setStreamsLoading(false);
        }
    };

    const loadMessages = async () => {
        if (!selectedStream) return;

        setLoading(true);
        try {
            const data = await api.listStreamMessages(selectedStream, subjectFilter || undefined);
            setMessages(data || []);
            setMessagesError(null);
        } catch (err) {
            setMessages([]);
            setMessagesError(err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (selectedStream) {
            loadMessages();
        }
    }, [selectedStream]);

    const handleSearch = () => {
        loadMessages();
    };

    // Filter messages by keyword in data or subject
    const filteredMessages = messages.filter((msg) => {
        if (!keywordFilter) return true;
        const keyword = keywordFilter.toLowerCase();
        return (
            msg.data.toLowerCase().includes(keyword) ||
            msg.subject.toLowerCase().includes(keyword)
        );
    });

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        Message Browser
                    </h1>
                    <p className="text-muted-foreground">View and inspect messages in streams</p>
                </div>
                <button
                    onClick={() => {
                        loadStreams();
                        loadMessages();
                    }}
                    className="glass glass-hover rounded-lg px-4 py-2 flex items-center gap-2 text-foreground"
                    disabled={loading}
                >
                    <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
                    Refresh
                </button>
            </div>

            {/* Stream selector and search */}
            <div className="glass rounded-xl p-6 mb-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Select Stream</label>
                        <select
                            value={selectedStream}
                            onChange={(e) => setSelectedStream(e.target.value)}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            {streams.map((stream) => (
                                <option key={stream.name} value={stream.name}>
                                    {stream.name} ({stream.messages} messages)
                                </option>
                            ))}
                        </select>
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Filter by Subject (optional)</label>
                        <div className="flex gap-2">
                            <input
                                type="text"
                                value={subjectFilter}
                                onChange={(e) => setSubjectFilter(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                                placeholder="e.g. events.user.*"
                                className="flex-1 bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            />
                            <button
                                onClick={handleSearch}
                                className="bg-primary text-primary-foreground rounded-lg px-4 py-2 hover:bg-primary/90 transition-colors flex items-center gap-2"
                            >
                                <Search className="w-4 h-4" />
                                Search
                            </button>
                        </div>
                    </div>
                </div>

                {/* Keyword search */}
                <div>
                    <label className="block text-sm font-medium mb-2 text-foreground">Search in Message Data</label>
                    <input
                        type="text"
                        value={keywordFilter}
                        onChange={(e) => setKeywordFilter(e.target.value)}
                        placeholder="Search by keyword in message content..."
                        className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    />
                    {keywordFilter && (
                        <p className="text-xs text-muted-foreground mt-1">
                            Showing {filteredMessages.length} of {messages.length} messages
                        </p>
                    )}
                </div>
            </div>

            {/* Messages list */}
            {streamsLoading && streams.length === 0 ? (
                <ListSkeleton />
            ) : streamsError ? (
                <ErrorState title="Couldn't load streams" error={streamsError} onRetry={loadStreams} />
            ) : streams.length === 0 ? (
                <EmptyState
                    icon={Database}
                    title="No streams to browse"
                    description="The browser shows messages stored in JetStream streams. Create a stream, publish to it, then inspect the messages here."
                >
                    <Link
                        href="/streams"
                        className="inline-block bg-primary text-primary-foreground rounded-lg px-6 py-3 hover:bg-primary/90 transition-colors"
                    >
                        Go to Streams
                    </Link>
                </EmptyState>
            ) : loading ? (
                <ListSkeleton />
            ) : messagesError ? (
                <ErrorState
                    title={`Couldn't load messages from "${selectedStream}"`}
                    error={messagesError}
                    onRetry={loadMessages}
                />
            ) : filteredMessages.length === 0 ? (
                <div className="glass rounded-xl p-12 text-center">
                    <Eye className="w-16 h-16 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-xl font-semibold mb-2 text-foreground">No messages found</h3>
                    <p className="text-muted-foreground">
                        {keywordFilter
                            ? `No messages match "${keywordFilter}"`
                            : selectedStream
                                ? 'This stream is empty or no messages match your filter'
                                : 'Select a stream to view messages'}
                    </p>
                </div>
            ) : (
                <div className="space-y-3">
                    {filteredMessages.map((message) => (
                        <MessageCard key={message.sequence} message={message} keyword={keywordFilter} />
                    ))}
                </div>
            )}
        </div>
    );
}

function MessageCard({ message, keyword }: { message: MessageDetail; keyword?: string }) {
    const [expanded, setExpanded] = useState(false);

    const copyToClipboard = useCopyToClipboard();

    const formatData = (data: string) => {
        try {
            const parsed = JSON.parse(data);
            return JSON.stringify(parsed, null, 2);
        } catch {
            return data;
        }
    };

    const isJSON = (data: string) => {
        try {
            JSON.parse(data);
            return true;
        } catch {
            return false;
        }
    };

    return (
        <div className="glass glass-hover rounded-lg p-4">
            <div className="flex items-center justify-between">
                <button
                    onClick={() => setExpanded(!expanded)}
                    className="flex-1 flex items-center gap-3 text-left"
                >
                    {expanded ? <ChevronUp className="w-4 h-4 text-primary" /> : <ChevronDown className="w-4 h-4 text-primary" />}
                    <div className="flex-1 grid grid-cols-3 gap-4">
                        <div>
                            <p className="text-xs text-muted-foreground">Sequence</p>
                            <p className="font-mono text-sm font-semibold text-foreground">#{message.sequence}</p>
                        </div>
                        <div>
                            <p className="text-xs text-muted-foreground">Subject</p>
                            <p className="font-mono text-sm text-foreground truncate">{message.subject}</p>
                        </div>
                        <div>
                            <p className="text-xs text-muted-foreground">Size</p>
                            <p className="text-sm text-foreground">{formatBytes(message.size)}</p>
                        </div>
                    </div>
                </button>
                <button
                    onClick={() => copyToClipboard(message.data)}
                    className="text-muted-foreground hover:text-foreground transition-colors p-2 rounded-lg hover:bg-card/50"
                    title="Copy data"
                >
                    <Copy className="w-4 h-4" />
                </button>
            </div>

            {expanded && (
                <div className="mt-4 pt-4 border-t border-border/50 space-y-3">
                    <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                            <span className="text-muted-foreground">Time:</span>
                            <span className="ml-2 text-foreground">{new Date(message.time).toLocaleString()}</span>
                        </div>
                        <div>
                            <span className="text-muted-foreground">Subject:</span>
                            <span className="ml-2 text-foreground font-mono">{message.subject}</span>
                        </div>
                    </div>

                    {message.headers && Object.keys(message.headers).length > 0 && (
                        <div className="glass rounded-lg p-3">
                            <p className="text-xs text-muted-foreground mb-2">Headers:</p>
                            <div className="space-y-1 text-sm">
                                {Object.entries(message.headers).map(([key, values]) => (
                                    <div key={key} className="flex gap-2">
                                        <span className="text-muted-foreground font-mono">{key}:</span>
                                        <span className="text-foreground">{values.join(', ')}</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    <div className="glass rounded-lg p-3">
                        <div className="flex items-center justify-between mb-2">
                            <p className="text-xs text-muted-foreground">Message Data:</p>
                            {isJSON(message.data) && (
                                <span className="text-xs text-green-500">JSON</span>
                            )}
                        </div>
                        <pre className="text-sm text-foreground font-mono whitespace-pre-wrap break-all overflow-auto max-h-96">
                            {formatData(message.data)}
                        </pre>
                    </div>
                </div>
            )}
        </div>
    );
}

function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}
