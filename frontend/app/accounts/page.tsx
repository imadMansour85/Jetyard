'use client';

import { useEffect, useState, useCallback } from 'react';
import { api, Account, AccountWithSeed } from '@/lib/api';
import { Users, Plus, Trash2, RefreshCw, Copy, Key, Download, AlertTriangle } from 'lucide-react';
import { useCopyToClipboard, useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { CardGridSkeleton, EmptyState, ErrorState } from '@/components/PageState';

export default function AccountsPage() {
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<unknown>(null);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const toast = useToast();
    const confirm = useConfirm();

    const loadAccounts = useCallback(async () => {
        try {
            const data = await api.listAccounts();
            setAccounts(data);
            setError(null);
        } catch (err) {
            setError(err);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadAccounts();
    }, [loadAccounts]);

    const handleDelete = async (name: string) => {
        const ok = await confirm({
            title: `Delete account "${name}"?`,
            message: 'The account and its signing key are removed from the dashboard. An account that still has users cannot be deleted.',
        });
        if (!ok) return;

        try {
            await api.deleteAccount(name);
            toast.success(`Account "${name}" deleted`);
            loadAccounts();
        } catch (err) {
            toast.error(`Failed to delete account "${name}"`, err);
        }
    };

    const copyToClipboard = useCopyToClipboard();

    return (
        <div className="animate-fade-in">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-foreground to-foreground/60 bg-clip-text text-transparent">
                        Accounts
                    </h1>
                    <p className="text-muted-foreground">Manage NATS accounts with NKey authentication</p>
                </div>
                <div className="flex gap-3">
                    <button
                        onClick={loadAccounts}
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
                        Create Account
                    </button>
                </div>
            </div>

            {loading ? (
                <CardGridSkeleton count={2} />
            ) : error ? (
                <ErrorState title="Couldn't load accounts" error={error} onRetry={loadAccounts} />
            ) : accounts.length === 0 ? (
                <EmptyState
                    icon={Users}
                    title="No accounts yet"
                    description="An account is an isolated namespace in NATS, signed by your operator key. Create one, then issue users in it from the Users page."
                    actionLabel="Create your first account"
                    onAction={() => setShowCreateModal(true)}
                />
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {accounts.map((account) => (
                        <div key={account.name} className="glass glass-hover rounded-xl p-6">
                            <div className="flex items-start justify-between mb-4">
                                <div>
                                    <h3 className="text-xl font-semibold mb-1 text-foreground">{account.name}</h3>
                                    <p className="text-xs text-muted-foreground">
                                        Created {new Date(account.created_at).toLocaleString()}
                                    </p>
                                </div>
                                <button
                                    onClick={() => handleDelete(account.name)}
                                    className="text-destructive hover:text-destructive/80 transition-colors p-2 hover:bg-destructive/10 rounded-lg"
                                    aria-label={`Delete account ${account.name}`}
                                >
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </div>

                            <div className="space-y-3">
                                {account.signing_key_missing && (
                                    <div className="rounded-lg p-3 bg-amber-500/10 border border-amber-500/20 flex gap-2 text-sm">
                                        <AlertTriangle className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                                        <p className="text-foreground">
                                            This account&apos;s signing key was not saved (it was created before
                                            seeds were persisted), so it can&apos;t issue new users. Existing users
                                            keep working. To issue users again, delete its users and re-create the
                                            account.
                                        </p>
                                    </div>
                                )}
                                <div className="glass rounded-lg p-3">
                                    <div className="flex items-center justify-between mb-2">
                                        <span className="text-xs text-muted-foreground flex items-center gap-1">
                                            <Key className="w-3 h-3" />
                                            Public Key
                                        </span>
                                        <button
                                            onClick={() => copyToClipboard(account.public_key)}
                                            className="text-muted-foreground hover:text-foreground transition-colors"
                                        >
                                            <Copy className="w-3 h-3" />
                                        </button>
                                    </div>
                                    <p className="text-xs font-mono text-foreground break-all">
                                        {account.public_key}
                                    </p>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {showCreateModal && (
                <CreateAccountModal
                    onClose={() => setShowCreateModal(false)}
                    onCreated={loadAccounts}
                />
            )}
        </div>
    );
}

function CreateAccountModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
    const [name, setName] = useState('');
    const [submitting, setSubmitting] = useState(false);
    const [createdAccount, setCreatedAccount] = useState<AccountWithSeed | null>(null);
    const toast = useToast();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSubmitting(true);

        try {
            const account = await api.createAccount({ name });
            setCreatedAccount(account);
        } catch (err) {
            toast.error('Failed to create account', err);
            setSubmitting(false);
        }
    };

    const downloadSeed = () => {
        if (!createdAccount) return;
        const blob = new Blob([createdAccount.seed], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${createdAccount.name}-account.nk`;
        a.click();
        URL.revokeObjectURL(url);
    };

    const copyToClipboard = useCopyToClipboard();

    const finish = () => {
        onCreated();
        onClose();
    };

    if (createdAccount) {
        return (
            <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
                <div className="glass rounded-xl p-6 max-w-2xl w-full">
                    <h2 className="text-2xl font-bold mb-2 text-foreground">Account Created!</h2>
                    <p className="text-amber-500 mb-6 text-sm">
                        ⚠️ Save these credentials now - they cannot be retrieved later!
                    </p>

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium mb-2 text-foreground">Account Name</label>
                            <div className="glass rounded-lg p-3">
                                <p className="text-foreground font-mono">{createdAccount.name}</p>
                            </div>
                        </div>

                        <div>
                            <div className="flex items-center justify-between mb-2">
                                <label className="block text-sm font-medium text-foreground">Public Key</label>
                                <button
                                    onClick={() => copyToClipboard(createdAccount.public_key)}
                                    className="text-primary hover:text-primary/80 transition-colors text-sm flex items-center gap-1"
                                >
                                    <Copy className="w-3 h-3" />
                                    Copy
                                </button>
                            </div>
                            <div className="glass rounded-lg p-3">
                                <p className="text-foreground font-mono text-sm break-all">{createdAccount.public_key}</p>
                            </div>
                        </div>

                        <div>
                            <div className="flex items-center justify-between mb-2">
                                <label className="block text-sm font-medium text-amber-500">Seed (Private Key)</label>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => copyToClipboard(createdAccount.seed)}
                                        className="text-primary hover:text-primary/80 transition-colors text-sm flex items-center gap-1"
                                    >
                                        <Copy className="w-3 h-3" />
                                        Copy
                                    </button>
                                    <button
                                        onClick={downloadSeed}
                                        className="text-primary hover:text-primary/80 transition-colors text-sm flex items-center gap-1"
                                    >
                                        <Download className="w-3 h-3" />
                                        Download
                                    </button>
                                </div>
                            </div>
                            <div className="glass rounded-lg p-3 bg-amber-500/10 border border-amber-500/20">
                                <p className="text-foreground font-mono text-sm break-all">{createdAccount.seed}</p>
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
                <h2 className="text-2xl font-bold mb-6 text-foreground">Create Account</h2>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2 text-foreground">Account Name</label>
                        <input
                            type="text"
                            required
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            className="w-full bg-input border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="my-account"
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
