'use client';

import { useEffect, useMemo, useState } from 'react';
import { api, type ClientConnection } from '@/lib/api';
import { RefreshCw, Radio, Search, AlertTriangle } from 'lucide-react';

const REFRESH_MS = 5000;

function formatStart(start?: string): string {
    if (!start) return 'N/A';
    const d = new Date(start);
    return isNaN(d.getTime()) ? start : d.toLocaleString();
}

export default function SubscriptionsPage() {
    const [connections, setConnections] = useState<ClientConnection[]>([]);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [query, setQuery] = useState('');

    const load = async (isInitial = false) => {
        isInitial ? setLoading(true) : setRefreshing(true);
        try {
            const data = await api.listConnections();
            setConnections(data);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load connections');
        } finally {
            isInitial ? setLoading(false) : setRefreshing(false);
        }
    };

    useEffect(() => {
        load(true);
        const id = setInterval(() => load(false), REFRESH_MS);
        return () => clearInterval(id);
    }, []);

    const filtered = useMemo(() => {
        const q = query.trim().toLowerCase();
        if (!q) return connections;
        return connections.filter((c) =>
            [c.name, c.ip, c.lang, String(c.cid), ...(c.subscriptions ?? [])]
                .some((v) => v?.toLowerCase().includes(q)),
        );
    }, [connections, query]);

    if (loading) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="animate-pulse-soft text-muted-foreground">Loading connections...</div>
            </div>
        );
    }

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        Subscriptions
                    </h1>
                    <p className="text-muted-foreground">Active client connections and what they're subscribed to</p>
                </div>
                <button
                    onClick={() => load(false)}
                    className="glass glass-hover rounded-lg px-4 py-2 flex items-center gap-2 text-foreground"
                    disabled={refreshing}
                >
                    <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
                    Refresh
                </button>
            </div>

            {error && (
                <div className="mb-4 flex items-center gap-2 p-3 rounded-lg text-sm bg-red-500/10 border border-red-500/30 text-red-400">
                    <AlertTriangle className="w-4 h-4 shrink-0" />
                    Could not load connections: {error}
                </div>
            )}

            <div className="relative mb-4">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                <input
                    type="text"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    placeholder="Filter by name, IP, language or subject"
                    className="w-full pl-9 pr-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-sm"
                />
            </div>

            <p className="text-xs text-muted-foreground mb-4">
                {filtered.length} of {connections.length} clients · refreshes every {REFRESH_MS / 1000}s
            </p>

            {filtered.length === 0 ? (
                <div className="glass rounded-xl p-6 text-center text-muted-foreground">
                    {connections.length === 0 ? 'No active client connections.' : 'No clients match this filter.'}
                </div>
            ) : (
                <div className="space-y-4">
                    {filtered.map((conn) => (
                        <div key={conn.cid} className="glass rounded-xl p-6">
                            <div className="flex items-center justify-between mb-3">
                                <div className="flex items-center gap-2">
                                    <Radio className="w-4 h-4 text-primary" />
                                    <span className="font-medium text-foreground">{conn.name || `Client #${conn.cid}`}</span>
                                </div>
                                <span className="text-sm font-mono text-muted-foreground">{conn.ip}:{conn.port}</span>
                            </div>

                            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm mb-3">
                                <div>
                                    <p className="text-muted-foreground">Language</p>
                                    <p className="text-foreground">{conn.lang || 'N/A'}</p>
                                </div>
                                <div>
                                    <p className="text-muted-foreground">Version</p>
                                    <p className="text-foreground">{conn.version || 'N/A'}</p>
                                </div>
                                <div>
                                    <p className="text-muted-foreground">Messages In / Out</p>
                                    <p className="text-foreground">{conn.in_msgs.toLocaleString()} / {conn.out_msgs.toLocaleString()}</p>
                                </div>
                                <div>
                                    <p className="text-muted-foreground">Connected Since</p>
                                    <p className="text-foreground">{formatStart(conn.start)}</p>
                                </div>
                            </div>

                            <div>
                                <p className="text-sm text-muted-foreground mb-1">Subscriptions ({conn.subscriptions?.length || 0})</p>
                                {conn.subscriptions && conn.subscriptions.length > 0 ? (
                                    <div className="flex flex-wrap gap-2">
                                        {conn.subscriptions.map((subj, i) => (
                                            <span key={i} className="px-2 py-1 rounded-md bg-muted text-xs font-mono text-foreground">
                                                {subj}
                                            </span>
                                        ))}
                                    </div>
                                ) : (
                                    <p className="text-xs text-muted-foreground">No subscriptions</p>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
