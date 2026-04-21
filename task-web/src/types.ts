export interface Task {
  id: string;
  title: string;
  description?: string;
  dueDate?: string;
  categoryId?: string | null;
  completed: boolean;
  createdAt: string;
  completedAt?: string | null;
  daysRemaining?: number | null;
  overdue?: boolean;
}

export interface Category {
  id: string;
  name: string;
}

export interface ApiError {
  error: string;
}

export interface User {
  email?: string;
  id?: string;
}

declare global {
  interface Window {
    __APP_CONFIG__?: {
      API_URL?: string;
    };
  }
}
