'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { api, type BackendVersion } from '@/lib/api';
import { Server, Radio, MessageSquare, Database, Home, Users, Shield, FolderKey, Activity, Eye, Settings, LogOut, Waypoints } from 'lucide-react';
import { clearToken } from '@/lib/auth';
import { useCurrentUser } from '@/components/AuthGuard';

const navigation = [
    { name: 'Dashboard', href: '/', icon: Home },
    { name: 'Server Info', href: '/server', icon: Activity },
    { name: 'Subscriptions', href: '/subscriptions', icon: Waypoints },
    { name: 'Streams', href: '/streams', icon: Database },
    { name: 'Consumers', href: '/consumers', icon: Server },
    { name: 'Messages', href: '/messages', icon: MessageSquare },
    { name: 'Browser', href: '/browser', icon: Eye },
    { name: 'KV Store', href: '/kv', icon: FolderKey },
    { name: 'Accounts', href: '/accounts', icon: Users },
    { name: 'Users', href: '/users', icon: Shield },
    { name: 'Settings', href: '/settings', icon: Settings },
];

export default function Sidebar() {
    const user = useCurrentUser();
    const [version, setVersion] = useState<BackendVersion | null>(null);

    useEffect(() => {
        // Purely informational: hide the footer line if it fails.
        api.getVersion().then(setVersion).catch(() => setVersion(null));
    }, []);

    const logout = () => {
        clearToken();
        window.location.href = '/login';
    };

    const pathname = usePathname();

    return (
        <div className="w-64 glass border-r border-border/50 flex flex-col">
            <div className="p-6 border-b border-border/50">
                <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center">
                        <Radio className="w-6 h-6 text-primary-foreground" />
                    </div>
                    <div>
                        <h1 className="text-xl font-bold text-foreground">Jetyard</h1>
                        <p className="text-xs text-muted-foreground">Dashboard for NATS</p>
                    </div>
                </div>
            </div>

            <nav className="flex-1 p-4 space-y-2">
                {navigation.map((item) => {
                    const isActive = pathname === item.href;
                    const Icon = item.icon;

                    return (
                        <Link
                            key={item.name}
                            href={item.href}
                            className={`
                flex items-center gap-3 px-4 py-3 rounded-lg transition-all duration-200
                ${isActive
                                    ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20'
                                    : 'text-muted-foreground hover:text-foreground hover:bg-card/50'
                                }
              `}
                        >
                            <Icon className="w-5 h-5" />
                            <span className="font-medium">{item.name}</span>
                        </Link>
                    );
                })}
            </nav>

            <div className="p-4 border-t border-border/50 space-y-3">
                <div className="glass rounded-lg p-4">
                    <div className="flex items-center gap-2 mb-2">
                        <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse-soft"></div>
                        <span className="text-xs font-medium text-muted-foreground">Connected</span>
                    </div>
                    <p className="text-xs text-muted-foreground">Real-time updates active</p>
                </div>

                <div className="flex items-center justify-between gap-2 px-1">
                    <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-foreground">{user?.username ?? '—'}</p>
                        <p className="text-xs text-muted-foreground">{user?.role ?? ''}</p>
                    </div>
                    <button
                        onClick={logout}
                        title="Sign out"
                        className="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-card/50 hover:text-foreground"
                    >
                        <LogOut className="h-4 w-4" />
                    </button>
                </div>

                {version && (
                    <p
                        className="px-1 text-[11px] text-muted-foreground/70 font-mono truncate"
                        title={[
                            `Backend ${version.version}`,
                            version.commit && `commit ${version.commit}`,
                            version.build_time && `built ${version.build_time}`,
                        ]
                            .filter(Boolean)
                            .join('\n')}
                    >
                        {version.version}
                        {version.commit && ` · ${version.commit.slice(0, 7)}`}
                    </p>
                )}
            </div>
        </div>
    );
}
