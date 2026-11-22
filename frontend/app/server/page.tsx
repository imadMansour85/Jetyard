'use client';

import { useEffect, useState } from 'react';
import { api, ServerInfo } from '@/lib/api';
import { Server, Cpu, HardDrive, Users, Zap, RefreshCw, Network } from 'lucide-react';
import { CardGridSkeleton, ErrorBanner, ErrorState } from '@/components/PageState';

export default function ServerPage() {
    const [serverInfo, setServerInfo] = useState<ServerInfo | null>(null);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [error, setError] = useState<unknown>(null);

    const loadServerInfo = async (isInitial = false) => {
        if (isInitial) {
            setLoading(true);
        } else {
            setRefreshing(true);
        }
        try {
            const data = await api.getServerInfo();
            setServerInfo(data);
            setError(null);
        } catch (err) {
            setError(err);
        } finally {
            if (isInitial) {
                setLoading(false);
            } else {
                setRefreshing(false);
            }
        }
    };

    useEffect(() => {
        loadServerInfo(true); // Initial load
        const interval = setInterval(() => loadServerInfo(false), 5000); // Refresh without loading state
        return () => clearInterval(interval);
    }, []);

    if (loading) {
        return <CardGridSkeleton count={4} />;
    }

    if (!serverInfo) {
        return (
            <ErrorState
                title="Couldn't load server info"
                error={
                    error ??
                    'No data from the NATS monitoring endpoint. Check that NATS runs with -m 8222 and that NATS_MONITOR_PORT matches.'
                }
                onRetry={() => loadServerInfo(true)}
            />
        );
    }

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        Server Info
                    </h1>
                    <p className="text-muted-foreground">NATS server health and statistics</p>
                </div>
                <button
                    onClick={() => loadServerInfo(false)}
                    className="glass glass-hover rounded-lg px-4 py-2 flex items-center gap-2 text-foreground"
                    disabled={refreshing}
                >
                    <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
                    Refresh
                </button>
            </div>

            {error ? <ErrorBanner message="Last refresh failed, showing older data" error={error} /> : null}

            {/* Server Identity */}
            <div className="glass rounded-xl p-6 mb-6">
                <h2 className="text-xl font-semibold mb-4 text-foreground flex items-center gap-2">
                    <Server className="w-5 h-5" />
                    Server Identity
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <InfoRow label="Server Name" value={serverInfo.server_name || 'N/A'} />
                    <InfoRow label="Server ID" value={serverInfo.server_id} mono />
                    <InfoRow label="Version" value={serverInfo.version} />
                    <InfoRow label="Go Version" value={serverInfo.go} />
                    <InfoRow label="Host" value={`${serverInfo.host}:${serverInfo.port}`} />
                    <InfoRow label="JetStream" value={serverInfo.jetstream ? 'Enabled' : 'Disabled'} />
                </div>
            </div>

            {/* Resource Usage */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
                <div className="glass rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-4">
                        <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center">
                            <Cpu className="w-6 h-6 text-white" />
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground">CPU Usage</p>
                            <p className="text-2xl font-bold text-foreground">{serverInfo.cpu.toFixed(1)}%</p>
                        </div>
                    </div>
                    <p className="text-xs text-muted-foreground">{serverInfo.cores} cores</p>
                </div>

                <div className="glass rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-4">
                        <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-purple-500 to-purple-600 flex items-center justify-center">
                            <HardDrive className="w-6 h-6 text-white" />
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground">Memory</p>
                            <p className="text-2xl font-bold text-foreground">{formatBytes(serverInfo.mem)}</p>
                        </div>
                    </div>
                    <p className="text-xs text-muted-foreground">Current usage</p>
                </div>

                <div className="glass rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-4">
                        <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-green-500 to-green-600 flex items-center justify-center">
                            <Users className="w-6 h-6 text-white" />
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground">Connections</p>
                            <p className="text-2xl font-bold text-foreground">{serverInfo.connections}</p>
                        </div>
                    </div>
                    <p className="text-xs text-muted-foreground">{serverInfo.total_connections} total</p>
                </div>
            </div>

            {/* Message Stats */}
            <div className="glass rounded-xl p-6 mb-6">
                <h2 className="text-xl font-semibold mb-4 text-foreground flex items-center gap-2">
                    <Zap className="w-5 h-5" />
                    Message Statistics
                </h2>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <StatCard label="Messages In" value={serverInfo.in_msgs.toLocaleString()} />
                    <StatCard label="Messages Out" value={serverInfo.out_msgs.toLocaleString()} />
                    <StatCard label="Bytes In" value={formatBytes(serverInfo.in_bytes)} />
                    <StatCard label="Bytes Out" value={formatBytes(serverInfo.out_bytes)} />
                </div>
            </div>

            {/* JetStream limits and usage */}
            {serverInfo.jetstream_info && (
                <div className="glass rounded-xl p-6 mb-6">
                    <h2 className="text-xl font-semibold mb-4 text-foreground flex items-center gap-2">
                        <HardDrive className="w-5 h-5" />
                        JetStream
                    </h2>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <StatCard label="Memory Used" value={formatBytes(serverInfo.jetstream_info.memory)} />
                        <StatCard label="Max Memory" value={formatLimit(serverInfo.jetstream_info.max_memory)} />
                        <StatCard label="Storage Used" value={formatBytes(serverInfo.jetstream_info.storage)} />
                        <StatCard label="Max Storage" value={formatLimit(serverInfo.jetstream_info.max_storage)} />
                    </div>
                    {serverInfo.jetstream_info.store_dir && (
                        <div className="mt-4">
                            <InfoRow label="Store Directory" value={serverInfo.jetstream_info.store_dir} mono />
                        </div>
                    )}
                </div>
            )}

            {/* Topology */}
            <div className="glass rounded-xl p-6">
                <h2 className="text-xl font-semibold mb-4 text-foreground flex items-center gap-2">
                    <Network className="w-5 h-5" />
                    Topology
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <InfoRow label="Routes" value={serverInfo.routes.toString()} />
                    <InfoRow label="Remotes" value={serverInfo.remotes.toString()} />
                    <InfoRow label="Leaf Nodes" value={serverInfo.leafnodes.toString()} />
                </div>
                {serverInfo.cluster && (
                    <div className="mt-4 pt-4 border-t border-border/50">
                        <p className="text-sm font-medium text-foreground mb-2">Cluster Info</p>
                        <div className="space-y-2">
                            <InfoRow label="Cluster Name" value={serverInfo.cluster.name} />
                            <InfoRow label="Cluster Address" value={serverInfo.cluster.addr} />
                        </div>
                    </div>
                )}
            </div>

            {/* System Info */}
            <div className="glass rounded-xl p-6 mt-6">
                <h2 className="text-xl font-semibold mb-4 text-foreground">System</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <InfoRow label="Uptime" value={serverInfo.uptime} />
                    <InfoRow label="Slow Consumers" value={serverInfo.slow_consumers.toString()} />
                    <InfoRow label="Max Payload" value={formatBytes(serverInfo.max_payload)} />
                </div>
            </div>
        </div>
    );
}

function InfoRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
    return (
        <div className="flex justify-between items-center">
            <span className="text-sm text-muted-foreground">{label}:</span>
            <span className={`text-sm font-medium text-foreground ${mono ? 'font-mono' : ''}`}>{value}</span>
        </div>
    );
}

function StatCard({ label, value }: { label: string; value: string }) {
    return (
        <div className="glass rounded-lg p-4">
            <p className="text-xs text-muted-foreground mb-1">{label}</p>
            <p className="text-xl font-bold text-foreground">{value}</p>
        </div>
    );
}

// JetStream reports a non-positive limit when none is configured.
function formatLimit(bytes: number): string {
    return bytes > 0 ? formatBytes(bytes) : 'Unlimited';
}

function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}
