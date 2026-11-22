import { authHeaders, clearToken, setToken, type AuthUser } from './auth';
import { ApiError } from './errors';

import { API_BASE } from './config';

export interface Stream {
    name: string;
    subjects: string[];
    retention: string;
    max_msgs: number;
    max_bytes: number;
    max_age: number;
    storage: string;
    replicas: number;
    messages: number;
    bytes: number;
    first_seq: number;
    last_seq: number;
}

export interface Consumer {
    stream_name: string;
    name: string;
    deliver_policy: string;
    ack_policy: string;
    ack_wait: number;
    max_deliver: number;
    filter_subject?: string;
    replay_policy: string;
    num_pending: number;
    num_ack_pending: number;
    num_redelivered: number;
    delivered: number;
}

export interface Message {
    subject: string;
    sequence: number;
    time: string;
    data: string;
    headers?: Record<string, string>;
}

export interface Stats {
    total_streams: number;
    total_consumers: number;
    total_messages: number;
    total_bytes: number;
    connected: boolean;
    server_info: string;
}

export interface Account {
    name: string;
    public_key: string;
    jwt: string;
    created_at: string;
    // True when the backend lost this account's seed (accounts created before
    // seeds were persisted). Such an account can no longer issue users.
    signing_key_missing?: boolean;
}

export interface AccountWithSeed extends Account {
    seed: string; // Only returned at creation
}

export interface CreateAccountRequest {
    name: string;
}

export interface Permission {
    allow?: string[];
    deny?: string[];
}

export interface Permissions {
    pub?: Permission;
    sub?: Permission;
}

export interface User {
    name: string;
    public_key: string;
    account_name: string;
    jwt: string;
    permissions?: Permissions;
    created_at: string;
}

export interface UserWithSeed extends User {
    seed: string; // Only returned at creation
    credentials_file: string;
}

export interface CreateUserRequest {
    name: string;
    account_name: string;
    permissions?: Permissions;
}

export interface KVBucket {
    name: string;
    description?: string;
    max_value_size: number;
    history: number;
    ttl: number;
    bucket: string;
    values: number;
    bytes: number;
    created_at: string;
}

export interface CreateKVBucketRequest {
    name: string;
    description?: string;
    history?: number;
    ttl?: number;
    max_value_size?: number;
}

export interface KVEntry {
    bucket: string;
    key: string;
    value: string;
    revision: number;
    created: string;
    operation: string;
}

export interface ServerInfo {
    server_id: string;
    server_name: string;
    version: string;
    proto: number;
    git_commit: string;
    go: string;
    host: string;
    port: number;
    max_payload: number;
    jetstream: boolean;
    jetstream_info?: {
        max_memory: number;
        max_storage: number;
        store_dir?: string;
        memory: number;
        storage: number;
        accounts: number;
    };
    now: string;
    uptime: string;
    mem: number;
    cores: number;
    cpu: number;
    connections: number;
    total_connections: number;
    routes: number;
    remotes: number;
    leafnodes: number;
    in_msgs: number;
    out_msgs: number;
    in_bytes: number;
    out_bytes: number;
    slow_consumers: number;
    cluster?: {
        name: string;
        addr: string;
        urls: string[];
    };
}

export interface NATSSettings {
    url: string;
    auth_type: 'none' | 'userpass' | 'token';
    username: string;
    password: string;
    token: string;
    tls_enabled: boolean;
    ca_cert: string;
    client_cert: string;
    client_key: string;
}

export interface TestConnectionResult {
    success: boolean;
    message: string;
}

export interface ClientConnection {
    cid: number;
    ip: string;
    port: number;
    name?: string;
    lang?: string;
    version?: string;
    start?: string;
    in_msgs: number;
    out_msgs: number;
    in_bytes: number;
    out_bytes: number;
    subscriptions?: string[];
}

export interface BackendVersion {
    version: string;
    commit?: string;
    build_time?: string;
    modified?: boolean;
    go_version: string;
}

export interface MessageDetail {
    subject: string;
    sequence: number;
    time: string;
    data: string;
    headers?: Record<string, string[]>;
    size: number;
}

async function fetchAPI(endpoint: string, options?: RequestInit) {
    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...authHeaders(),
            ...options?.headers,
        },
    });

    if (response.status === 401 && typeof window !== 'undefined') {
        clearToken();
        if (!window.location.pathname.startsWith('/login')) {
            window.location.href = '/login';
        }
        throw new Error('Session expired');
    }

    if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new ApiError(
            body.error || `Request failed (${response.status})`,
            response.status,
            body.request_id || response.headers.get('X-Request-Id') || undefined,
        );
    }

    return response.json();
}

export const api = {
    // Build info of the backend (public endpoint)
    getVersion: async (): Promise<BackendVersion> => {
        const res = await fetch(`${API_BASE}/version`);
        if (!res.ok) throw new ApiError(`Request failed (${res.status})`, res.status);
        return res.json();
    },

    // Stats
    getStats: (): Promise<Stats> => fetchAPI('/api/stats'),

    // Streams
    listStreams: async (): Promise<Stream[]> => {
        const result = await fetchAPI('/api/streams');
        return result || [];
    },
    getStream: (name: string): Promise<Stream> => fetchAPI(`/api/streams/${name}`),
    createStream: (data: {
        name: string;
        subjects: string[];
        retention?: string;
        max_msgs?: number;
        max_bytes?: number;
        max_age?: number;
        storage?: string;
        replicas?: number;
    }): Promise<Stream> =>
        fetchAPI('/api/streams', {
            method: 'POST',
            body: JSON.stringify(data),
        }),
    deleteStream: (name: string): Promise<void> =>
        fetchAPI(`/api/streams/${name}`, { method: 'DELETE' }),

    // Consumers
    listConsumers: async (streamName: string): Promise<Consumer[]> => {
        const result = await fetchAPI(`/api/consumers?stream=${streamName}`);
        return result || [];
    },
    createConsumer: (data: {
        stream_name: string;
        name: string;
        deliver_policy?: string;
        ack_policy?: string;
        ack_wait?: number;
        max_deliver?: number;
        filter_subject?: string;
        replay_policy?: string;
    }): Promise<Consumer> =>
        fetchAPI('/api/consumers', {
            method: 'POST',
            body: JSON.stringify(data),
        }),
    deleteConsumer: (streamName: string, name: string): Promise<void> =>
        fetchAPI(`/api/consumers/${name}?stream=${streamName}`, { method: 'DELETE' }),

    // Messages
    publishMessage: (data: {
        subject: string;
        data: string;
        headers?: Record<string, string>;
    }): Promise<void> =>
        fetchAPI('/api/publish', {
            method: 'POST',
            body: JSON.stringify(data),
        }),
    getMessages: async (streamName: string, limit = 10): Promise<Message[]> => {
        const result = await fetchAPI(`/api/messages/${streamName}?limit=${limit}`);
        return result || [];
    },

    // Accounts
    listAccounts: async (): Promise<Account[]> => {
        const result = await fetchAPI('/api/accounts');
        return result || [];
    },
    getAccount: (name: string): Promise<Account> => fetchAPI(`/api/accounts/${name}`),
    createAccount: (data: CreateAccountRequest): Promise<AccountWithSeed> =>
        fetchAPI('/api/accounts', {
            method: 'POST',
            body: JSON.stringify(data),
        }),
    deleteAccount: (name: string): Promise<void> =>
        fetchAPI(`/api/accounts/${name}`, { method: 'DELETE' }),
    getAccountJWT: (name: string): Promise<string> =>
        fetch(`${API_BASE}/api/accounts/${name}/jwt`, { headers: authHeaders() }).then(r => r.text()),

    // Users
    listUsers: async (accountName?: string): Promise<User[]> => {
        const url = accountName ? `/api/users?account=${accountName}` : '/api/users';
        const result = await fetchAPI(url);
        return result || [];
    },
    getUser: (name: string): Promise<User> => fetchAPI(`/api/users/${name}`),
    createUser: (data: CreateUserRequest): Promise<UserWithSeed> =>
        fetchAPI('/api/users', {
            method: 'POST',
            body: JSON.stringify(data),
        }),
    deleteUser: (name: string): Promise<void> =>
        fetchAPI(`/api/users/${name}`, { method: 'DELETE' }),
    updateUserPermissions: (name: string, permissions: Permissions): Promise<User> =>
        fetchAPI(`/api/users/${name}/permissions`, {
            method: 'PUT',
            body: JSON.stringify(permissions),
        }),

    // KV Buckets
    listKVBuckets: async (): Promise<KVBucket[]> => {
        const result = await fetchAPI('/api/kv/buckets');
        return result || [];
    },
    getKVBucket: (name: string): Promise<KVBucket> => fetchAPI(`/api/kv/buckets/${name}`),
    createKVBucket: (data: CreateKVBucketRequest): Promise<KVBucket> =>
        fetchAPI('/api/kv/buckets', {
            method: 'POST',
            body: JSON.stringify(data),
        }),
    deleteKVBucket: (name: string): Promise<void> =>
        fetchAPI(`/api/kv/buckets/${name}`, { method: 'DELETE' }),

    // KV Keys
    listKeys: async (bucket: string): Promise<string[]> => {
        const result = await fetchAPI(`/api/kv/buckets/${bucket}/keys`);
        return result || [];
    },
    getKey: (bucket: string, key: string): Promise<KVEntry> =>
        fetchAPI(`/api/kv/buckets/${bucket}/keys/${key}`),
    putKey: (bucket: string, key: string, value: string): Promise<void> =>
        fetchAPI(`/api/kv/buckets/${bucket}/keys/${key}`, {
            method: 'PUT',
            body: JSON.stringify({ value }),
        }),
    deleteKey: (bucket: string, key: string): Promise<void> =>
        fetchAPI(`/api/kv/buckets/${bucket}/keys/${key}`, { method: 'DELETE' }),

    // Server Info
    getServerInfo: (): Promise<ServerInfo> => fetchAPI('/api/server'),

    // Connections & subscriptions
    listConnections: async (): Promise<ClientConnection[]> => {
        const result = await fetchAPI('/api/connections');
        return result || [];
    },

    // Message Browser
    listStreamMessages: (stream: string, subject?: string, start?: number, limit?: number): Promise<MessageDetail[]> => {
        const params = new URLSearchParams();
        if (subject) params.append('subject', subject);
        if (start) params.append('start', start.toString());
        if (limit) params.append('limit', limit.toString());
        const query = params.toString();
        return fetchAPI(`/api/streams/${stream}/messages${query ? `?${query}` : ''}`);
    },
    getMessageBySeq: (stream: string, seq: number): Promise<MessageDetail> =>
        fetchAPI(`/api/streams/${stream}/messages/${seq}`),

    // Dashboard authentication
    needsSetup: async (): Promise<boolean> => {
        const res = await fetch(`${API_BASE}/api/auth/setup`);
        if (!res.ok) throw new Error('Backend unreachable');
        const data = await res.json();
        return data.needs_setup;
    },
    login: async (username: string, password: string): Promise<AuthUser> => {
        const res = await fetch(`${API_BASE}/api/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });
        const data = await res.json().catch(() => ({ error: 'Login failed' }));
        if (!res.ok) throw new Error(data.error || 'Login failed');
        setToken(data.token);
        return data.user;
    },
    registerFirstAdmin: async (username: string, password: string): Promise<AuthUser> => {
        const res = await fetch(`${API_BASE}/api/auth/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });
        const data = await res.json().catch(() => ({ error: 'Registration failed' }));
        if (!res.ok) throw new Error(data.error || 'Registration failed');
        setToken(data.token);
        return data.user;
    },
    me: (): Promise<AuthUser> => fetchAPI('/api/auth/me'),
    changePassword: (currentPassword: string, newPassword: string): Promise<void> =>
        fetchAPI('/api/auth/change-password', {
            method: 'POST',
            body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
        }),
    listDashboardUsers: (): Promise<AuthUser[]> => fetchAPI('/api/auth/users'),
    createDashboardUser: (username: string, password: string, role: string): Promise<AuthUser> =>
        fetchAPI('/api/auth/users', {
            method: 'POST',
            body: JSON.stringify({ username, password, role }),
        }),
    deleteDashboardUser: (username: string): Promise<void> =>
        fetchAPI(`/api/auth/users/${username}`, { method: 'DELETE' }),

    // NATS connection settings
    getNATSSettings: (): Promise<NATSSettings> => fetchAPI('/api/settings/nats'),
    testNATSConnection: (settings: NATSSettings): Promise<TestConnectionResult> =>
        fetchAPI('/api/settings/nats/test', {
            method: 'POST',
            body: JSON.stringify(settings),
        }),
    saveNATSSettings: (settings: NATSSettings): Promise<TestConnectionResult> =>
        fetchAPI('/api/settings/nats/save', {
            method: 'POST',
            body: JSON.stringify(settings),
        }),
};
