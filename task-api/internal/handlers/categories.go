package handlers

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/asdlc/task-api/internal/httputil"
	"github.com/asdlc/task-api/internal/middleware"
	"github.com/asdlc/task-api/internal/models"
	"github.com/asdlc/task-api/internal/store"
	"github.com/asdlc/task-api/internal/validation"
	"github.com/google/uuid"
)

type CategoryHandler struct {
	Store store.Store
}

func NewCategoryHandler(st store.Store) *CategoryHandler {
	return &CategoryHandler{Store: st}
}

type categoryRequest struct {
	Name string `json:"name"`
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	cats := h.Store.ListCategories(userID)
	sort.Slice(cats, func(i, j int) bool { return cats[i].Name < cats[j].Name })
	httputil.WriteJSON(w, http.StatusOK, cats)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	var req categoryRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := validation.SanitizeString(req.Name)
	if name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(name) > 100 {
		httputil.WriteError(w, http.StatusBadRequest, "name too long")
		return
	}
	c := &models.Category{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.Store.CreateCategory(c); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			httputil.WriteError(w, http.StatusConflict, "category name already exists")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create category")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, c)
}

func (h *CategoryHandler) getOwnedCategory(w http.ResponseWriter, r *http.Request, id string) (*models.Category, bool) {
	userID := middleware.UserID(r)
	c, err := h.Store.GetCategory(id)
	if err != nil || c.UserID != userID {
		httputil.WriteError(w, http.StatusNotFound, "category not found")
		return nil, false
	}
	return c, true
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	c, ok := h.getOwnedCategory(w, r, id)
	if !ok {
		return
	}
	var req categoryRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := validation.SanitizeString(req.Name)
	if name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(name) > 100 {
		httputil.WriteError(w, http.StatusBadRequest, "name too long")
		return
	}
	updated := *c
	updated.Name = name
	if err := h.Store.UpdateCategory(&updated); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			httputil.WriteError(w, http.StatusConflict, "category name already exists")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update category")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, updated)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	userID := middleware.UserID(r)
	if _, ok := h.getOwnedCategory(w, r, id); !ok {
		return
	}
	if err := h.Store.DeleteCategory(id); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete category")
		return
	}
	h.Store.UnassignCategory(userID, id)
	w.WriteHeader(http.StatusNoContent)
}
