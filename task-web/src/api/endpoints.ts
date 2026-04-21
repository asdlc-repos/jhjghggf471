import { api } from './client';
import { Task, Category } from '../types';

export const authApi = {
  register: (email: string, password: string) =>
    api.post<void>('/auth/register', { email, password }),
  login: (email: string, password: string) =>
    api.post<void>('/auth/login', { email, password }),
  logout: () => api.post<void>('/auth/logout'),
  passwordReset: (email: string) => api.post<void>('/auth/password-reset', { email }),
  passwordResetConfirm: (token: string, newPassword: string) =>
    api.post<void>('/auth/password-reset/confirm', { token, newPassword }),
};

export interface TaskFilters {
  categoryId?: string;
  dueFrom?: string;
  dueTo?: string;
}

function buildQuery(f: TaskFilters): string {
  const p = new URLSearchParams();
  if (f.categoryId) p.set('categoryId', f.categoryId);
  if (f.dueFrom) p.set('dueFrom', f.dueFrom);
  if (f.dueTo) p.set('dueTo', f.dueTo);
  const s = p.toString();
  return s ? `?${s}` : '';
}

export const tasksApi = {
  list: (filters: TaskFilters = {}) => api.get<Task[]>(`/tasks${buildQuery(filters)}`),
  create: (input: Partial<Task>) => api.post<Task>('/tasks', input),
  update: (id: string, input: Partial<Task>) => api.put<Task>(`/tasks/${id}`, input),
  remove: (id: string) => api.del<void>(`/tasks/${id}`),
  complete: (id: string) => api.post<Task>(`/tasks/${id}/complete`),
  incomplete: (id: string) => api.post<Task>(`/tasks/${id}/incomplete`),
};

export const categoriesApi = {
  list: () => api.get<Category[]>('/categories'),
  create: (name: string) => api.post<Category>('/categories', { name }),
  update: (id: string, name: string) => api.put<Category>(`/categories/${id}`, { name }),
  remove: (id: string) => api.del<void>(`/categories/${id}`),
};
