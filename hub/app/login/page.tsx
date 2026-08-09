"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/use-auth";
import { ApiRequestError, api } from "@/lib/api-client";
import { getErrorMessage } from "@/lib/error-messages";

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [version, setVersion] = useState<string | null>(null);
  const emailRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    emailRef.current?.focus();
  }, []);

  useEffect(() => {
    let active = true;
    api
      .getVersion()
      .then((v) => {
        if (active) setVersion(v.version);
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await login(email, password);
      router.push("/dashboard");
    } catch (err) {
      if (err instanceof ApiRequestError && (err.status === 401 || err.status === 400)) {
        setError("Invalid email or password. Double-check your email and password.");
      } else {
        const copy = getErrorMessage(err);
        setError(copy.hint ? `${copy.message} ${copy.hint}` : copy.message);
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)] px-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">RenOS</h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-2">
            Sign in to your device mesh
          </p>
        </div>

        <form onSubmit={handleSubmit} className="card space-y-4">
          {error && (
            <div
              id="login-error"
              role="alert"
              className="flex items-center gap-2 p-2 rounded-sm bg-[rgba(255,59,48,0.08)] text-body-small text-[var(--color-error)]"
            >
              {error}
            </div>
          )}

          <div>
            <label htmlFor="email" className="block text-body-small text-[var(--color-text-secondary)] mb-1">
              Email
            </label>
            <input
              ref={emailRef}
              id="email"
              name="email"
              type="email"
              className="input"
              placeholder="admin@renos.local"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
              aria-required="true"
              aria-describedby={error ? "login-error" : undefined}
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-body-small text-[var(--color-text-secondary)] mb-1">
              Password
            </label>
            <input
              id="password"
              name="password"
              type="password"
              className="input"
              placeholder="Enter your password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
              aria-required="true"
              aria-describedby={error ? "login-error" : undefined}
            />
          </div>

          <button type="submit" className="btn-primary w-full" disabled={loading}>
            {loading ? "Signing in..." : "Sign in"}
          </button>
        </form>

        <p className="text-caption text-[var(--color-text-tertiary)] text-center mt-4">
          {version ? `RenOS v${version}` : "RenOS"}
        </p>
      </div>
    </div>
  );
}
