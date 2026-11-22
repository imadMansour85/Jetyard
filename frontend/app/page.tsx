'use client';

import { useEffect, useState, useCallback } from 'react';
import { api, Stats } from '@/lib/api';
import { useSSE, SSEEvent } from '@/lib/sse';
import { Database, Server, MessageSquare, HardDrive, TrendingUp, Users, Shield, FolderKey } from 'lucide-react';
import { CardGridSkeleton, ErrorBanner } from '@/components/PageState';

export default function Dashboard() {
    const [stats, setStats] = useState<Stats | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<unknown>(null);

    const loadStats = useCallback(async () => {
        try {
            const data = await api.getStats();
            setStats(data);
            setError(null);
        } catch (err) {
            setError(err);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadStats();
        const interval = setInterval(loadStats, 5000);
        return () => clearInterval(interval);
    }, [loadStats]);

    const handleSSEEvent = useCallback((event: SSEEvent) => {
        // Reload stats on any change
        if (event.type !== 'connected') {
            loadStats();
        }
    }, [loadStats]);

    const { connected } = useSSE(handleSSEEvent);

    if (loading) {
        return <CardGridSkeleton count={4} />;
    }

    const statCards = [
        {
            label: 'Total Streams',
            value: stats?.total_streams || 0,
            icon: Database,
            color: 'from-blue-500 to-blue-600',
        },
        {
            label: 'Total Consumers',
            value: stats?.total_consumers || 0,
            icon: Server,
            color: 'from-purple-500 to-purple-600',
        },
        {
            label: 'Total Messages',
            value: stats?.total_messages || 0,
            icon: MessageSquare,
            color: 'from-green-500 to-green-600',
        },
        {
            label: 'Total Bytes',
            value: formatBytes(stats?.total_bytes || 0),
            icon: HardDrive,
            color: 'from-orange-500 to-orange-600',
        },
    ];

    return (
        <div className="animate-fade-in">
            <div className="mb-8">
                <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                    Dashboard
                </h1>
                <div className="flex items-center gap-2 text-muted-foreground">
                    <div className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'} animate-pulse-soft`}></div>
                    <p className="text-sm">
                        {connected ? 'Live updates active' : 'Reconnecting...'}
                        {stats?.server_info && ` • ${stats.server_info}`}
                    </p>
                </div>
            </div>

            {error ? (
                <ErrorBanner
                    message={stats ? 'Last refresh failed, showing older numbers' : "Couldn't load stats"}
                    error={error}
                />
            ) : null}

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                {statCards.map((stat, index) => {
                    const Icon = stat.icon;
                    return (
                        <div
                            key={stat.label}
                            className="glass glass-hover rounded-xl p-6 animate-slide-up"
                            style={{ animationDelay: `${index * 50}ms` }}
                        >
                            <div className="flex items-start justify-between mb-4">
                                <div className={`w-12 h-12 rounded-lg bg-gradient-to-br ${stat.color} flex items-center justify-center`}>
                                    <Icon className="w-6 h-6 text-white" />
                                </div>
                                <TrendingUp className="w-4 h-4 text-green-500" />
                            </div>
                            <h3 className="text-3xl font-bold mb-1 text-foreground">{stat.value}</h3>
                            <p className="text-sm text-muted-foreground">{stat.label}</p>
                        </div>
                    );
                })}
            </div>

            <div className="glass rounded-xl p-6">
                <h2 className="text-xl font-semibold mb-4">Quick Actions</h2>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <a
                        href="/streams"
                        className="glass-hover rounded-lg p-4 flex items-center gap-4 group"
                    >
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                            <Database className="w-5 h-5 text-primary" />
                        </div>
                        <div>
                            <h3 className="font-medium text-foreground">Manage Streams</h3>
                            <p className="text-sm text-muted-foreground">Create and configure streams</p>
                        </div>
                    </a>

                    <a
                        href="/consumers"
                        className="glass-hover rounded-lg p-4 flex items-center gap-4 group"
                    >
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                            <Server className="w-5 h-5 text-primary" />
                        </div>
                        <div>
                            <h3 className="font-medium text-foreground">Manage Consumers</h3>
                            <p className="text-sm text-muted-foreground">Configure message consumers</p>
                        </div>
                    </a>

                    <a
                        href="/messages"
                        className="glass-hover rounded-lg p-4 flex items-center gap-4 group"
                    >
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                            <MessageSquare className="w-5 h-5 text-primary" />
                        </div>
                        <div>
                            <h3 className="font-medium text-foreground">Publish Messages</h3>
                            <p className="text-sm text-muted-foreground">Send messages to subjects</p>
                        </div>
                    </a>

                    <a
                        href="/accounts"
                        className="glass-hover rounded-lg p-4 flex items-center gap-4 group"
                    >
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                            <Users className="w-5 h-5 text-primary" />
                        </div>
                        <div>
                            <h3 className="font-medium text-foreground">Manage Accounts</h3>
                            <p className="text-sm text-muted-foreground">Configure NATS accounts</p>
                        </div>
                    </a>

                    <a
                        href="/users"
                        className="glass-hover rounded-lg p-4 flex items-center gap-4 group"
                    >
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                            <Shield className="w-5 h-5 text-primary" />
                        </div>
                        <div>
                            <h3 className="font-medium text-foreground">Manage Users</h3>
                            <p className="text-sm text-muted-foreground">Configure user access</p>
                        </div>
                    </a>
                </div>
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
