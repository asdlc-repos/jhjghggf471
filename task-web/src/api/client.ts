const DEFAULT_API = '/api';

export function getApiBase(): string {
  const runtime = typeof window !== 'undefined' ? window.__APP_CONFIG__?.API_URL : '';
  if (runtime && runtime.length > 0) return runtime.replace(/\/$/, '');
  const viteUrl = import.meta.env.VITE_API_URL as string | undefined;
  if (viteUrl && viteUrl.length > 0) return viteUrl.replace(/\/$/, '');
  return DEFAULT_API;
}

export class ApiError extends Error {
  status: number;
  body: any;
  constructor(status: number, message: string, body?: any) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const base = getApiBase();
  const url = `${base}${path}`;
  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...(options.headers as Record<string, string> | undefined),
  };
  if (options.body && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json';
  }
  let res: Response;
  try {
    res = await fetch(url, {
      ...options,
      headers,
      credentials: 'include',
    });
  } catch (e: any) {
    throw new ApiError(0, 'Network error. Please check your connection.');
  }

  if (res.status === 204) {
    return undefined as unknown as T;
  }

  let data: any = null;
  const text = await res.text();
  if (text.length > 0) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { raw: text };
    }
  }

  if (!res.ok) {
    const msg = (data && typeof data === 'object' && data.error) || defaultMessage(res.status);
    throw new ApiError(res.status, msg, data);
  }
  return data as T;
}

function defaultMessage(status: number): string {
  switch (status) {
    case 400:
      return 'Invalid request.';
    case 401:
      return 'Not authenticated.';
    case 403:
      return 'Forbidden.';
    case 404:
      return 'Not found.';
    case 409:
      return 'Conflict.';
    case 423:
      return 'Account locked. Please try again later.';
    case 500:
      return 'Server error. Please try again.';
    default:
      return `Request failed (${status}).`;
  }
}

export const api = {
  get: <T>(p: string) => request<T>(p, { method: 'GET' }),
  post: <T>(p: string, body?: any) =>
    request<T>(p, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(p: string, body?: any) =>
    request<T>(p, { method: 'PUT', body: body ? JSON.stringify(body) : undefined }),
  del: <T>(p: string) => request<T>(p, { method: 'DELETE' }),
};
