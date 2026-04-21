import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { authApi } from '../api/endpoints';
import { ApiError } from '../api/client';

interface AuthState {
  isAuthenticated: boolean;
  email: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  markUnauthenticated: () => void;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

const STORAGE_KEY = 'task-web:auth';

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [email, setEmail] = useState<string | null>(() => {
    try {
      return window.localStorage.getItem(STORAGE_KEY);
    } catch {
      return null;
    }
  });
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    try {
      if (email) window.localStorage.setItem(STORAGE_KEY, email);
      else window.localStorage.removeItem(STORAGE_KEY);
    } catch {
      /* ignore */
    }
  }, [email]);

  const login = useCallback(async (em: string, pw: string) => {
    setLoading(true);
    try {
      await authApi.login(em, pw);
      setEmail(em);
    } finally {
      setLoading(false);
    }
  }, []);

  const register = useCallback(async (em: string, pw: string) => {
    setLoading(true);
    try {
      await authApi.register(em, pw);
      try {
        await authApi.login(em, pw);
        setEmail(em);
      } catch (e) {
        if (e instanceof ApiError) {
          /* leave login to user */
        } else {
          throw e;
        }
      }
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      /* ignore */
    }
    setEmail(null);
  }, []);

  const markUnauthenticated = useCallback(() => {
    setEmail(null);
  }, []);

  const value: AuthState = {
    isAuthenticated: !!email,
    email,
    loading,
    login,
    register,
    logout,
    markUnauthenticated,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
