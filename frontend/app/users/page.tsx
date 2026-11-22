'use client';

import { useEffect, useState, useCallback } from 'react';
import { api, User, UserWithSeed, Account, Permissions } from '@/lib/api';
import { User as UserIcon, Plus, Trash2, RefreshCw, Copy, Key, Download, Shield } from 'lucide-react';
import Link from 'next/link';
import { useCopyToClipboard, useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { CardGridSkeleton, EmptyState, ErrorBanner, ErrorState } from '@/components/PageState';

export default function UsersPage() {
    const [users, setUsers] = useState<User[]>([]);
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<unknown>(null);
    const [accountsLoaded, setAccountsLoaded] = useState(false);
    const [accountsError, setAccountsError] = useState<unknown>(null);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const toast = useToast();
    const confirm = useConfirm();
    const copyToClipboard = useCopyToClipboard();

    const loadUsers = useCallback(async () => {
        try {
            const data = await api.listUsers();
            setUsers(data);
            setError(null);
        } catch (err) {
            setError(err);
        } finally {
            setLoading(false);
        }
    }, []);

    const loadAccounts = useCallback(async () => {
        try {
            const data = await api.listAccounts();
            setAccounts(data);
            setAccountsError(null);
        } catch (err) {
            setAccountsError(err);
        } finally {
            setAccountsLoaded(true);
        }
    }, []);

    const reload = useCallback(() => {
        loadUsers();
        loadAccounts();
    }, [loadUsers, loadAccounts]);

    useEffect(() => {
        reload();
    }, [reload]);

    const handleDelete = async (name: string) => {
        const ok = await confirm({
            title: `Delete user "${name}"?`,
            message: 'The user is removed from the dashboard, but this does not revoke its .creds file: NATS keeps accepting it until the user is revoked in the account JWT.',
        });
        if (!ok) return;

        try {
            await api.deleteUser(name);
            toast.success(`User "${name}" deleted`);
            loadUsers();
        } catch (err) {
            toast.error(`Failed to delete user "${name}"`, err);
        }
    };

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        Users
                    </h1>
                    <p className="text-muted-foreground">Manage NATS users with NKey authentication</p>
                </div>
                <div className="flex gap-3">
                    <button
                        onClick={reload}
                        className="glass glass-hover rounded-lg px-4 py-2 flex items-center gap-2 text-foreground"
                    >
                        <RefreshCw className="w-4 h-4" />
                        Refresh
                    </button>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="bg-primary text-primary-foreground rounded-lg px-4 py-2 flex items-center gap-2 hover:bg-primary/90 transition-colors shadow-lg shadow-primary/20 disabled:opacity-50"
                        disabled={accounts.length === 0}
                    >
                        <Plus className="w-4 h-4" />
                        Create User
                    </button>
                </div>
            </div>

            {accountsError ? (
                <ErrorBanner message="Couldn't load accounts, so new users can't be created" error={accountsError} />
            ) : (
                accountsLoaded &&
                accounts.length === 0 && (
                    <div className="glass rounded-xl p-4 mb-6 border border-amber-500/20 bg-amber-500/10 text-sm text-amber-500">
                        Users belong to an account. <Link href="/accounts" className="underline">Create an account</Link>{' '}
                        first, then come back to issue users.
                    </div>
                )
            )}

            {loading ? (
                <CardGridSkeleton />
            ) : error ? (
                <ErrorState title="Couldn't load users" error={error} onRetry={reload} />
            ) : users.length === 0 ? (
                <EmptyState
                    icon={UserIcon}
                    title="No users yet"
                    description="A user gets a .creds file that applications use to connect to NATS, with optional publish and subscribe permissions."
                    actionLabel={accounts.length > 0 ? 'Create your first user' : undefined}
                    onAction={accounts.length > 0 ? () => setShowCreateModal(true) : undefined}
                />
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {users.map((user) => (
                        <div key={user.name} className="glass glass-hover rounded-xl p-6">
                            <div className="flex items-start justify-between mb-4">
                                <div>
                                    <h3 className="text-xl font-semibold mb-1 text-foreground">{user.name}</h3>
                                    <p className="text-xs text-muted-foreground">
                                        Account: {user.account_name}
                                    </p>
                                </div>
                                <button
                                    onClick={() => handleDelete(user.name)}
                                    className="text-destructive hover:text-destructive/80 transition-colors p-2 hover:bg-destructive/10 rounded-lg"
                                    aria-label={`Delete user ${user.name}`}
                                >
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </div>

                            <div className="space-y-3">
                                <div className="glass rounded-lg p-3">
                                    <div className="flex items-center justify-between mb-2">
                                        <span className="text-xs text-muted-foreground flex items-center gap-1">
                                            <Key className="w-3 h-3" />
                                            Public Key
                                        </span>
                                        <button
                                            onClick={() => copyToClipboard(user.public_key)}
                                            className="text-muted-foreground hover:text-foreground transition-colors"
                                        >
                                            <Copy className="w-3 h-3" />
                                        </button>
                                    </div>
                                    <p className="text-xs font-mono text-foreground break-all">
                                        {user.public_key}
                                    </p>
                                </div>

                                {user.permissions && (
                                    <div className="glass rounded-lg p-3">
                                        <div className="flex items-center gap-1 mb-2">
                                            <Shield className="w-3 h-3 text-muted-foreground" />
                                            <span className="text-xs text-muted-foreground">Permissions</span>
                                        </div>
                                        <div className="space-y-1 text-xs">
                                            {user.permissions.pub?.allow && (
                                                <div>
                                                    <span className="text-green-500">Publish:</span>{' '}
                                                    <span className="text-foreground">
                                                        {user.permissions.pub.allow.join(', ')}
                                                    </span>
                                                </div>
                                            )}
                                            {user.permissions.sub?.allow && (
                                                <div>
                                                    <span className="text-blue-500">Subscribe:</span>{' '}
                                                    <span className="text-foreground">
                                                        {user.permissions.sub.allow.join(', ')}
                                                    </span>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {showCreateModal && (
                <CreateUserModal
                    accounts={accounts}
                    onClose={() => setShowCreateModal(false)}
                    onCreated={loadUsers}
                />
            )}
        </div>
    );
}

function CreateUserModal({
    accounts,
    onClose,
    onCreated,
}: {
    accounts: Account[];
    onClose: () => void;
    onCreated: () => void;
}) {
    const [formData, setFormData] = useState({
        name: '',
        account_name: accounts[0]?.name || '',
        pub_allow: '',
        sub_allow: '',
    });
    const [submitting, setSubmitting] = useState(false);
    const [createdUser, setCreatedUser] = useState<UserWithSeed | null>(null);
    const toast = useToast();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSubmitting(true);

        try {
            const permissions: Permissions = {};
            if (formData.pub_allow) {
                permissions.pub = {
                    allow: formData.pub_allow.split(',').map(s => s.trim()),
                };
            }
            if (formData.sub_allow) {
                permissions.sub = {
                    allow: formData.sub_allow.split(',').map(s => s.trim()),
                };
            }

            const user = await api.createUser({
                name: formData.name,
                account_name: formData.account_name,
                permissions: Object.keys(permissions).length > 0 ? permissions : undefined,
            });
            setCreatedUser(user);
        } catch (err) {
            toast.error('Failed to create user', err);
            setSubmitting(false);
        }
    };

    const downloadCredentials = () => {
        if (!createdUser) return;
        const blob = new Blob([createdUser.credentials_file], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${createdUser.name}.creds`;
        a.click();
        URL.revokeObjectURL(url);
    };

    const copyToClipboard = useCopyToClipboard();

    const finish = () => {
        onCreated();
        onClose();
    };

    if (createdUser) {
        return (
            <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
                <div className="glass rounded-xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
                    <h2 className="text-2xl font-bold mb-2 text-foreground">User Created!</h2>
                    <p className="text-amber-500 mb-6 text-sm">
                        ⚠️ Save these credentials now - they cannot be retrieved later!
                    </p>

                    <div className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className="block text-sm font-medium mb-2 text-foreground">User Name</label>
                                <div className="glass rounded-lg p-3">
                                    <p className="text-foreground font-mono text-sm">{createdUser.name}</p>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-2 text-foreground">Account</label>
                                <div className="glass rounded-lg p-3">
                                    <p className="text-foreground font-mono text-sm">{createdUser.account_name}</p>
                                </div>
                            </div>
                        </div>

                        <div>
                            <div className="flex items-center justify-between mb-2">
                                <label className="block text-sm font-medium text-foreground">Public Key</label>
                                <button
                                    onClick={() => copyToClipboard(createdUser.public_key)}
                                    className="text-primary hover:text-primary/80 transition-colors text-sm flex items-center gap-1"
                                >
                                    <Copy className="w-3 h-3" />
                                    Copy
                                </button>
                            </div>
                            <div className="glass rounded-lg p-3">
                                <p className="text-foreground font-mono text-xs break-all">{createdUser.public_key}</p>
                            </div>
                        </div>

                        <div>
                            <div className="flex items-center justify-between mb-2">
                                <label className="block text-sm font-medium text-amber-500">Seed (Private Key)</label>
                                <button
                                    onClick={() => copyToClipboard(createdUser.seed)}
                                    className="text-primary hover:text-primary/80 transition-colors text-sm flex items-center gap-1"
                                >
                                    <Copy className="w-3 h-3" />
                                    Copy
                                </button>
                            </div>
                            <div className="glass rounded-lg p-3 bg-amber-500/10 border border-amber-500/20">
                                <p className="text-foreground font-mono text-xs break-all">{createdUser.seed}</p>
                            </div>
                        </div>

                        <div className="glass rounded-lg p-4 bg-primary/10 border border-primary/20">
                            <div className="flex items-center justify-between mb-2">
                                <div>
                                    <p className="text-sm font-semibold text-foreground">Credentials File</p>
                                    <p className="text-xs text-muted-foreground">
                                        Download and use  with NATS clients
                                    </p>
                                </div>
                                <button
                                    onClick={downloadCredentials}
                                    className="bg-primary text-primary-foreground rounded-lg px-4 py-2 hover:bg-primary/90 transition-colors flex items-center gap-2"
                                >
                                    <Download className="w-4 h-4" />
                                    Download .creds
                                </button>
                            </div>
                        </div>
                    </div>

                    <div className="flex gap-3 pt-6">
                        <button
                            onClick={finish}
                            className="flex-1 bg-primary text-primary-foreground rounded-lg px-4 py-2 hover:bg-primary/90 transition-colors"
                        >
                            Done
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
            <div className="glass rounded-xl p-6 max-w-lg w-full">
                <h2 className="text-2xl font-bold mb-6 text-foreground">Create User</h2>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">User Name</label>
                        <input
                            type="text"
                            required
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="my-user"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Account</label>
                        <select
                            required
                            value={formData.account_name}
                            onChange={(e) => setFormData({ ...formData, account_name: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            {accounts.map((account) => (
                                <option key={account.name} value={account.name}>
                                    {account.name}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">
                            Publish Subjects (comma-separated, optional)
                        </label>
                        <input
                            type="text"
                            value={formData.pub_allow}
                            onChange={(e) => setFormData({ ...formData, pub_allow: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="events.*, orders.>"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">
                            Subscribe Subjects (comma-separated, optional)
                        </label>
                        <input
                            type="text"
                            value={formData.sub_allow}
                            onChange={(e) => setFormData({ ...formData, sub_allow: e.target.value })}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="events.*, orders.>"
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
