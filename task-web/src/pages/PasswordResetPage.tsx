import { FormEvent, useState } from 'react';
import { Link } from 'react-router-dom';
import { authApi } from '../api/endpoints';
import { extractMessage } from '../utils/errors';
import Spinner from '../components/Spinner';

type Mode = 'request' | 'confirm';

export default function PasswordResetPage() {
  const [mode, setMode] = useState<Mode>('request');
  const [email, setEmail] = useState('');
  const [token, setToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function requestReset(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setMessage(null);
    setSubmitting(true);
    try {
      await authApi.passwordReset(email.trim());
      setMessage('If an account exists, a reset link has been sent. Check the server logs for the token in development.');
      setMode('confirm');
    } catch (err) {
      setError(extractMessage(err, 'Could not start password reset.'));
    } finally {
      setSubmitting(false);
    }
  }

  async function confirmReset(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setMessage(null);
    if (newPassword.length < 8) {
      setError('New password must be at least 8 characters.');
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('Passwords do not match.');
      return;
    }
    setSubmitting(true);
    try {
      await authApi.passwordResetConfirm(token.trim(), newPassword);
      setMessage('Password updated. You can now log in with your new password.');
    } catch (err) {
      setError(extractMessage(err, 'Reset failed. Check your token and try again.'));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <h1>Password reset</h1>
        <p className="auth-sub">
          {mode === 'request'
            ? 'Enter your email and we will send a reset token.'
            : 'Enter the reset token you received and set a new password.'}
        </p>

        <div className="mode-toggle" role="tablist" aria-label="Reset mode">
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'request'}
            className={`mode-tab ${mode === 'request' ? 'active' : ''}`}
            onClick={() => setMode('request')}
          >
            Request
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'confirm'}
            className={`mode-tab ${mode === 'confirm' ? 'active' : ''}`}
            onClick={() => setMode('confirm')}
          >
            Confirm
          </button>
        </div>

        {mode === 'request' ? (
          <form onSubmit={requestReset} noValidate>
            <label className="field">
              <span className="field-label">Email</span>
              <input
                type="email"
                autoComplete="email"
                required
                autoFocus
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>
            {error && <div className="alert alert-error" role="alert">{error}</div>}
            {message && <div className="alert alert-success" role="status">{message}</div>}
            <button type="submit" className="btn btn-primary btn-block" disabled={submitting}>
              {submitting ? <Spinner label="Sending" /> : 'Send reset token'}
            </button>
          </form>
        ) : (
          <form onSubmit={confirmReset} noValidate>
            <label className="field">
              <span className="field-label">Reset token</span>
              <input
                type="text"
                required
                autoFocus
                value={token}
                onChange={(e) => setToken(e.target.value)}
              />
            </label>
            <label className="field">
              <span className="field-label">New password</span>
              <input
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </label>
            <label className="field">
              <span className="field-label">Confirm new password</span>
              <input
                type="password"
                autoComplete="new-password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </label>
            {error && <div className="alert alert-error" role="alert">{error}</div>}
            {message && <div className="alert alert-success" role="status">{message}</div>}
            <button type="submit" className="btn btn-primary btn-block" disabled={submitting}>
              {submitting ? <Spinner label="Updating" /> : 'Update password'}
            </button>
          </form>
        )}

        <div className="auth-links">
          <Link to="/login">Back to login</Link>
        </div>
      </div>
    </div>
  );
}
