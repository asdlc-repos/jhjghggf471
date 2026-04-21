import { FormEvent, useCallback, useEffect, useState } from 'react';
import { categoriesApi } from '../api/endpoints';
import { ApiError } from '../api/client';
import { Category } from '../types';
import { useAuth } from '../context/AuthContext';
import { useToast } from '../context/ToastContext';
import { extractMessage } from '../utils/errors';
import Spinner from '../components/Spinner';

export default function CategoriesPage() {
  const toast = useToast();
  const { markUnauthenticated } = useAuth();
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [newName, setNewName] = useState('');
  const [creating, setCreating] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState('');

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

  const load = useCallback(async () => {
    setLoading(true);
    setErr(null);
    try {
      const list = await categoriesApi.list();
      setCategories(Array.isArray(list) ? list : []);
    } catch (e) {
      if (handleAuthError(e)) return;
      setErr(extractMessage(e, 'Could not load categories.'));
    } finally {
      setLoading(false);
    }
  }, [handleAuthError]);

  useEffect(() => {
    load();
  }, [load]);

  async function createCategory(e: FormEvent) {
    e.preventDefault();
    const name = newName.trim();
    if (!name) return;
    setCreating(true);
    const tempId = `temp-${Date.now()}`;
    const optimistic: Category = { id: tempId, name };
    setCategories((c) => [...c, optimistic]);
    setNewName('');
    try {
      const created = await categoriesApi.create(name);
      if (created && created.id) {
        setCategories((c) => c.map((x) => (x.id === tempId ? created : x)));
      } else {
        await load();
      }
      toast.success('Category created.');
    } catch (e) {
      setCategories((c) => c.filter((x) => x.id !== tempId));
      if (handleAuthError(e)) return;
      if (e instanceof ApiError && e.status === 409) {
        toast.error('A category with that name already exists.');
      } else {
        toast.error(extractMessage(e, 'Could not create category.'));
      }
    } finally {
      setCreating(false);
    }
  }

  function startEdit(c: Category) {
    setEditingId(c.id);
    setEditingName(c.name);
  }

  function cancelEdit() {
    setEditingId(null);
    setEditingName('');
  }

  async function saveEdit(c: Category) {
    const name = editingName.trim();
    if (!name || name === c.name) {
      cancelEdit();
      return;
    }
    const prev = c.name;
    setCategories((cs) => cs.map((x) => (x.id === c.id ? { ...x, name } : x)));
    setEditingId(null);
    setEditingName('');
    try {
      const updated = await categoriesApi.update(c.id, name);
      if (updated && updated.id) {
        setCategories((cs) => cs.map((x) => (x.id === c.id ? updated : x)));
      }
      toast.success('Category renamed.');
    } catch (e) {
      setCategories((cs) => cs.map((x) => (x.id === c.id ? { ...x, name: prev } : x)));
      if (handleAuthError(e)) return;
      if (e instanceof ApiError && e.status === 409) {
        toast.error('A category with that name already exists.');
      } else {
        toast.error(extractMessage(e, 'Could not rename category.'));
      }
    }
  }

  async function deleteCategory(c: Category) {
    if (!confirm(`Delete category "${c.name}"? Tasks will become uncategorized.`)) return;
    const snapshot = categories;
    setCategories((cs) => cs.filter((x) => x.id !== c.id));
    try {
      await categoriesApi.remove(c.id);
      toast.success('Category deleted.');
    } catch (e) {
      setCategories(snapshot);
      if (!handleAuthError(e)) toast.error(extractMessage(e, 'Could not delete category.'));
    }
  }

  return (
    <div className="categories-page">
      <section className="page-header">
        <div>
          <h1>Categories</h1>
          <p className="page-sub">Organize your tasks into groups.</p>
        </div>
      </section>

      <form className="inline-form" onSubmit={createCategory}>
        <label className="field grow">
          <span className="field-label">New category name</span>
          <input
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="e.g. Work"
          />
        </label>
        <button type="submit" className="btn btn-primary" disabled={creating || !newName.trim()}>
          {creating ? <Spinner label="Adding" /> : 'Add category'}
        </button>
      </form>

      {err && <div className="alert alert-error" role="alert">{err}</div>}

      {loading ? (
        <div className="loading-row"><Spinner label="Loading categories" /></div>
      ) : categories.length === 0 ? (
        <div className="empty-state">
          <p>No categories yet. Create one above.</p>
        </div>
      ) : (
        <ul className="category-list">
          {categories.map((c) => (
            <li key={c.id} className="category-row">
              {editingId === c.id ? (
                <form
                  className="inline-form"
                  onSubmit={(e) => {
                    e.preventDefault();
                    saveEdit(c);
                  }}
                >
                  <input
                    type="text"
                    value={editingName}
                    autoFocus
                    onChange={(e) => setEditingName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Escape') {
                        e.preventDefault();
                        cancelEdit();
                      }
                    }}
                  />
                  <button type="submit" className="btn btn-primary">Save</button>
                  <button type="button" className="btn btn-secondary" onClick={cancelEdit}>Cancel</button>
                </form>
              ) : (
                <>
                  <span className="category-name">{c.name}</span>
                  <div className="category-actions">
                    <button type="button" className="btn btn-secondary" onClick={() => startEdit(c)}>
                      Rename
                    </button>
                    <button type="button" className="btn btn-danger" onClick={() => deleteCategory(c)}>
                      Delete
                    </button>
                  </div>
                </>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
