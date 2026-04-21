import { ApiError } from '../api/client';

export function extractMessage(e: unknown, fallback = 'Something went wrong.'): string {
  if (e instanceof ApiError) return e.message;
  if (e instanceof Error) return e.message;
  return fallback;
}
