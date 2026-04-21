import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { categoriesApi, tasksApi, TaskFilters } from '../api/endpoints';
import { ApiError } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useToast } from '../context/ToastContext';
import { Category, Task } from '../types';
import { daysUntil, humanDaysRemaining, isOverdue } from '../utils/dates';
import { extractMessage } from '../utils/errors';
import Modal from '../components/Modal';
import Spinner from '../components/Spinner';

type EditorState = { mode: 'create' } | { mode: 'edit'; task: Task } | null;

export default function DashboardPage() {
  const toast = useToast();
  const { markUnauthenticated } = useAuth();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [filters, setFilters] = useState<TaskFilters>({});
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [editor, setEditor] = useState<EditorState>(null);

  const handleAuthError = useCallback(
    (e: unknown) => {
      if (e instanceof ApiError && e.status === 401) {
        markUnauthenticated();
        return true;
      }
      return false;
    },
    [markUnauthenticated]
  );

  const loadCategories = useCallback(async () => {
    try {
      const cats = await categoriesApi.list();
      setCategories(Array.isArray(cats) ? cats : []);
    } catch (e) {
      if (handleAuthError(e)) return;
      /* categories optional */
    }
  }, [handleAuthError]);

  const loadTasks = useCallback(
    async (f: TaskFilters) => {
      setLoading(true);
      setErr(null);
      try {
        const list = await tasksApi.list(f);
        setTasks(Array.isArray(list) ? list : []);
      } catch (e) {
        if (handleAuthError(e)) return;
        setErr(extractMessage(e, 'Could not load tasks.'));
      } finally {
        setLoading(false);
      }
    },
    [handleAuthError]
  );

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  useEffect(() => {
    loadTasks(filters);
  }, [loadTasks, filters]);

  const categoryName = useCallback(
    (id?: string | null) => categories.find((c) => c.id === id)?.name || 'Uncategorized',
    [categories]
  );

  const sortedTasks = useMemo(() => {
    return [...tasks].sort((a, b) => {
      if (a.completed !== b.completed) return a.completed ? 1 : -1;
      const ao = isOverdue(a), bo = isOverdue(b);
      if (ao !== bo) return ao ? -1 : 1;
      const ad = a.dueDate ? daysUntil(a.dueDate) ?? 9999 : 9999;
      const bd = b.dueDate ? daysUntil(b.dueDate) ?? 9999 : 9999;
      if (ad !== bd) return ad - bd;
      return (a.createdAt || '').localeCompare(b.createdAt || '');
    });
  }, [tasks]);

  async function toggleComplete(task: Task) {
    const prev = task.completed;
    // optimistic update
    setTasks((cur) =>
      cur.map((t) =>
        t.id === task.id ? { ...t, completed: !prev, completedAt: !prev ? new Date().toISOString() : null } : t
      )
    );
    try {
      const updated = prev ? await tasksApi.incomplete(task.id) : await tasksApi.complete(task.id);
      if (updated && updated.id) {
        setTasks((cur) => cur.map((t) => (t.id === task.id ? updated : t)));
      }
    } catch (e) {
      setTasks((cur) => cur.map((t) => (t.id === task.id ? { ...t, completed: prev } : t)));
      if (!handleAuthError(e)) toast.error(extractMessage(e, 'Could not update task.'));
    }
  }

  async function deleteTask(task: Task) {
    if (!confirm(`Delete task "${task.title}"?`)) return;
    const snapshot = tasks;
    setTasks((cur) => cur.filter((t) => t.id !== task.id));
    try {
      await tasksApi.remove(task.id);
      toast.success('Task deleted.');
    } catch (e) {
      setTasks(snapshot);
      if (!handleAuthError(e)) toast.error(extractMessage(e, 'Could not delete task.'));
    }
  }

  async function saveTask(values: TaskFormValues) {
    if (editor?.mode === 'edit') {
      const target = editor.task;
      const optimistic: Task = { ...target, ...values };
      setTasks((cur) => cur.map((t) => (t.id === target.id ? optimistic : t)));
      try {
        const updated = await tasksApi.update(target.id, values);
        if (updated && updated.id) {
          setTasks((cur) => cur.map((t) => (t.id === target.id ? updated : t)));
        }
        toast.success('Task updated.');
        setEditor(null);
      } catch (e) {
        setTasks((cur) => cur.map((t) => (t.id === target.id ? target : t)));
        if (!handleAuthError(e)) toast.error(extractMessage(e, 'Could not update task.'));
      }
    } else {
      const tempId = `temp-${Date.now()}`;
      const now = new Date().toISOString();
      const optimistic: Task = {
        id: tempId,
        title: values.title,
        description: values.description,
        dueDate: values.dueDate,
        categoryId: values.categoryId,
        completed: false,
        createdAt: now,
      };
      setTasks((cur) => [optimistic, ...cur]);
      try {
        const created = await tasksApi.create(values);
        if (created && created.id) {
          setTasks((cur) => cur.map((t) => (t.id === tempId ? created : t)));
        } else {
          await loadTasks(filters);
        }
        toast.success('Task created.');
        setEditor(null);
      } catch (e) {
        setTasks((cur) => cur.filter((t) => t.id !== tempId));
        if (!handleAuthError(e)) toast.error(extractMessage(e, 'Could not create task.'));
      }
    }
  }

  return (
    <div className="dashboard">
      <section className="page-header">
        <div>
          <h1>Your tasks</h1>
          <p className="page-sub">
            {tasks.length} total · {tasks.filter((t) => isOverdue(t)).length} overdue ·{' '}
            {tasks.filter((t) => t.completed).length} completed
          </p>
        </div>
        <button
          type="button"
          className="btn btn-primary"
          onClick={() => setEditor({ mode: 'create' })}
        >
          + New task
        </button>
      </section>

      <section className="filters" aria-label="Task filters">
        <label className="field inline">
          <span className="field-label">Category</span>
          <select
            value={filters.categoryId || ''}
            onChange={(e) =>
              setFilters((f) => ({ ...f, categoryId: e.target.value || undefined }))
            }
          >
            <option value="">All categories</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>
        <label className="field inline">
          <span className="field-label">Due from</span>
          <input
            type="date"
            value={filters.dueFrom || ''}
            onChange={(e) => setFilters((f) => ({ ...f, dueFrom: e.target.value || undefined }))}
          />
        </label>
        <label className="field inline">
          <span className="field-label">Due to</span>
          <input
            type="date"
            value={filters.dueTo || ''}
            onChange={(e) => setFilters((f) => ({ ...f, dueTo: e.target.value || undefined }))}
          />
        </label>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={() => setFilters({})}
          disabled={!filters.categoryId && !filters.dueFrom && !filters.dueTo}
        >
          Clear filters
        </button>
        <Link to="/categories" className="btn btn-link">
          Manage categories →
        </Link>
      </section>

      {err && <div className="alert alert-error" role="alert">{err}</div>}

      {loading ? (
        <div className="loading-row"><Spinner label="Loading tasks" /></div>
      ) : sortedTasks.length === 0 ? (
        <div className="empty-state">
          <p>No tasks match your filters.</p>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => setEditor({ mode: 'create' })}
          >
            Create your first task
          </button>
        </div>
      ) : (
        <ul className="task-list">
          {sortedTasks.map((task) => {
            const overdue = isOverdue(task);
            return (
              <li
                key={task.id}
                className={`task-card ${task.completed ? 'completed' : ''} ${overdue ? 'overdue' : ''}`}
              >
                <label className="task-check">
                  <input
                    type="checkbox"
                    checked={task.completed}
                    onChange={() => toggleComplete(task)}
                    aria-label={task.completed ? 'Mark incomplete' : 'Mark complete'}
                  />
                </label>
                <div className="task-body">
                  <div className="task-title-row">
                    <h3 className="task-title">{task.title}</h3>
                    {overdue && <span className="badge badge-danger">Overdue</span>}
                    <span className="badge badge-muted">{categoryName(task.categoryId)}</span>
                  </div>
                  {task.description && <p className="task-desc">{task.description}</p>}
                  <div className="task-meta">
                    {task.dueDate && (
                      <span>
                        Due <strong>{task.dueDate}</strong>
                        {' · '}
                        <span className={overdue ? 'text-danger' : ''}>
                          {humanDaysRemaining(task.dueDate)}
                        </span>
                      </span>
                    )}
                  </div>
                </div>
                <div className="task-actions">
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={() => setEditor({ mode: 'edit', task })}
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    className="btn btn-danger"
                    onClick={() => deleteTask(task)}
                  >
                    Delete
                  </button>
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {editor && (
        <Modal
          title={editor.mode === 'edit' ? 'Edit task' : 'New task'}
          onClose={() => setEditor(null)}
        >
          <TaskForm
            categories={categories}
            initial={editor.mode === 'edit' ? editor.task : undefined}
            onCancel={() => setEditor(null)}
            onSubmit={saveTask}
          />
        </Modal>
      )}
    </div>
  );
}

interface TaskFormValues {
  title: string;
  description?: string;
  dueDate?: string;
  categoryId?: string | null;
}

function TaskForm({
  categories,
  initial,
  onCancel,
  onSubmit,
}: {
  categories: Category[];
  initial?: Task;
  onCancel: () => void;
  onSubmit: (v: TaskFormValues) => void | Promise<void>;
}) {
  const [title, setTitle] = useState(initial?.title || '');
  const [description, setDescription] = useState(initial?.description || '');
  const [dueDate, setDueDate] = useState(initial?.dueDate || '');
  const [categoryId, setCategoryId] = useState(initial?.categoryId || '');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = title.trim();
    if (!trimmed) {
      setError('Title is required.');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await onSubmit({
        title: trimmed,
        description: description.trim() || undefined,
        dueDate: dueDate || undefined,
        categoryId: categoryId || null,
      });
    } catch (err) {
      setError(extractMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} noValidate>
      <label className="field">
        <span className="field-label">Title</span>
        <input
          type="text"
          required
          autoFocus
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
      </label>
      <label className="field">
        <span className="field-label">Description</span>
        <textarea
          rows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </label>
      <div className="field-grid">
        <label className="field">
          <span className="field-label">Due date</span>
          <input
            type="date"
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
          />
        </label>
        <label className="field">
          <span className="field-label">Category</span>
          <select
            value={categoryId || ''}
            onChange={(e) => setCategoryId(e.target.value)}
          >
            <option value="">Uncategorized</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>
      </div>
      {error && <div className="alert alert-error" role="alert">{error}</div>}
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="btn btn-primary" disabled={submitting}>
          {submitting ? <Spinner label="Saving" /> : initial ? 'Save changes' : 'Create task'}
        </button>
      </div>
    </form>
  );
}
