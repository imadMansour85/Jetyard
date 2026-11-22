'use client';

import { useEffect, useState } from 'react';
import { Settings, Wifi, WifiOff, Save, FlaskConical, Lock, User, Key, ShieldCheck } from 'lucide-react';
import { api, type NATSSettings } from '@/lib/api';

const defaultSettings: NATSSettings = {
  url: 'nats://localhost:4222',
  auth_type: 'none',
  username: '',
  password: '',
  token: '',
  tls_enabled: false,
  ca_cert: '',
  client_cert: '',
  client_key: '',
};

export default function SettingsPage() {
  const [settings, setSettings] = useState<NATSSettings>(defaultSettings);
  const [loading, setLoading] = useState(true);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<{ success: boolean; message: string } | null>(null);

  useEffect(() => {
    api.getNATSSettings()
      .then(data => {
        setSettings(s => ({ ...s, ...data, password: '', token: '', client_key: '' }));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function set<K extends keyof NATSSettings>(key: K, value: NATSSettings[K]) {
    setSettings(s => ({ ...s, [key]: value }));
    setTestResult(null);
    setSaveResult(null);
  }

  async function testConnection() {
    setTesting(true);
    setTestResult(null);
    try {
      const data = await api.testNATSConnection(settings);
      setTestResult(data);
    } catch (err) {
      setTestResult({ success: false, message: err instanceof Error ? err.message : 'Could not reach the backend.' });
    } finally {
      setTesting(false);
    }
  }

  async function saveSettings() {
    setSaving(true);
    setSaveResult(null);
    try {
      const data = await api.saveNATSSettings(settings);
      setSaveResult(data);
    } catch (err) {
      setSaveResult({ success: false, message: err instanceof Error ? err.message : 'Could not reach the backend.' });
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
          <Settings className="w-5 h-5 text-primary" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-foreground">Settings</h1>
          <p className="text-sm text-muted-foreground">Configure your NATS server connection</p>
        </div>
      </div>

      {/* Connection card */}
      <div className="glass rounded-xl border border-border/50 p-6 space-y-6">
        <div className="flex items-center gap-2 pb-2 border-b border-border/50">
          <Wifi className="w-4 h-4 text-primary" />
          <h2 className="font-semibold text-foreground">NATS Connection</h2>
        </div>

        {/* Server URL */}
        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">Server URL</label>
          <input
            type="text"
            value={settings.url}
            onChange={e => set('url', e.target.value)}
            placeholder="nats://localhost:4222"
            className="w-full px-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-sm font-mono"
          />
        </div>

        {/* Auth type */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">Authentication</label>
          <div className="flex gap-2">
            {(['none', 'userpass', 'token'] as const).map(type => (
              <button
                key={type}
                onClick={() => set('auth_type', type)}
                className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium border transition-colors ${
                  settings.auth_type === type
                    ? 'bg-primary text-primary-foreground border-primary'
                    : 'bg-background text-muted-foreground border-border hover:border-primary/50'
                }`}
              >
                {type === 'none' && <Lock className="w-3.5 h-3.5" />}
                {type === 'userpass' && <User className="w-3.5 h-3.5" />}
                {type === 'token' && <Key className="w-3.5 h-3.5" />}
                {type === 'none' ? 'None' : type === 'userpass' ? 'User / Password' : 'Token'}
              </button>
            ))}
          </div>
        </div>

        {/* User/pass fields */}
        {settings.auth_type === 'userpass' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Username</label>
              <input
                type="text"
                value={settings.username}
                onChange={e => set('username', e.target.value)}
                placeholder="admin"
                className="w-full px-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-sm"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Password</label>
              <input
                type="password"
                value={settings.password}
                onChange={e => set('password', e.target.value)}
                placeholder="••••••••"
                className="w-full px-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-sm"
              />
            </div>
          </div>
        )}

        {/* Token field */}
        {settings.auth_type === 'token' && (
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Auth Token</label>
            <input
              type="password"
              value={settings.token}
              onChange={e => set('token', e.target.value)}
              placeholder="your-nats-auth-token"
              className="w-full px-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-sm font-mono"
            />
          </div>
        )}

        {/* TLS toggle */}
        <div className="flex items-center justify-between p-3 rounded-lg bg-background border border-border">
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-muted-foreground" />
            <span className="text-sm font-medium text-foreground">TLS / Secure Connection</span>
          </div>
          <button
            onClick={() => set('tls_enabled', !settings.tls_enabled)}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              settings.tls_enabled ? 'bg-primary' : 'bg-muted'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
                settings.tls_enabled ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
        </div>

        {/* TLS certs */}
        {settings.tls_enabled && (
          <div className="space-y-4 p-4 rounded-lg border border-border/50 bg-background/50">
            <p className="text-xs text-muted-foreground">Paste PEM-encoded certificates below. Leave empty if not needed.</p>

            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">CA Certificate</label>
              <textarea
                value={settings.ca_cert}
                onChange={e => set('ca_cert', e.target.value)}
                placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                rows={4}
                className="w-full px-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-xs font-mono resize-none"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Client Certificate</label>
              <textarea
                value={settings.client_cert}
                onChange={e => set('client_cert', e.target.value)}
                placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                rows={4}
                className="w-full px-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-xs font-mono resize-none"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Client Private Key</label>
              <textarea
                value={settings.client_key}
                onChange={e => set('client_key', e.target.value)}
                placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
                rows={4}
                className="w-full px-3 py-2 rounded-lg bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 text-xs font-mono resize-none"
              />
            </div>
          </div>
        )}

        {/* Test result */}
        {testResult && (
          <div className={`flex items-center gap-2 p-3 rounded-lg text-sm ${
            testResult.success
              ? 'bg-green-500/10 border border-green-500/30 text-green-400'
              : 'bg-red-500/10 border border-red-500/30 text-red-400'
          }`}>
            {testResult.success ? <Wifi className="w-4 h-4 shrink-0" /> : <WifiOff className="w-4 h-4 shrink-0" />}
            {testResult.message}
          </div>
        )}

        {/* Save result */}
        {saveResult && (
          <div className={`flex items-center gap-2 p-3 rounded-lg text-sm ${
            saveResult.success
              ? 'bg-green-500/10 border border-green-500/30 text-green-400'
              : 'bg-red-500/10 border border-red-500/30 text-red-400'
          }`}>
            {saveResult.success ? <Save className="w-4 h-4 shrink-0" /> : <WifiOff className="w-4 h-4 shrink-0" />}
            {saveResult.message}
          </div>
        )}

        {/* Action buttons */}
        <div className="flex gap-3 pt-2">
          <button
            onClick={testConnection}
            disabled={testing || !settings.url}
            className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-card/50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <FlaskConical className="w-4 h-4" />
            {testing ? 'Testing…' : 'Test Connection'}
          </button>

          <button
            onClick={saveSettings}
            disabled={saving || !settings.url}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-primary/20"
          >
            <Save className="w-4 h-4" />
            {saving ? 'Saving…' : 'Save & Reconnect'}
          </button>
        </div>
      </div>

      {/* Info box */}
      <div className="glass rounded-xl border border-border/50 p-4 text-sm text-muted-foreground space-y-1">
        <p className="font-medium text-foreground">How it works</p>
        <p>Settings are saved to <code className="text-xs bg-muted px-1 py-0.5 rounded">/data/nats-settings.json</code> inside the container.</p>
        <p>After saving, the backend restarts automatically and reconnects with the new configuration.</p>
      </div>
    </div>
  );
}
