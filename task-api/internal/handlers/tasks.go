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

type TaskHandler struct {
	Store store.Store
}

func NewTaskHandler(st store.Store) *TaskHandler {
	return &TaskHandler{Store: st}
}

type taskResponse struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	DueDate        string     `json:"dueDate,omitempty"`
	CategoryID     string     `json:"categoryId,omitempty"`
	Completed      bool       `json:"completed"`
	CreatedAt      time.Time  `json:"createdAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	DaysRemaining  *int       `json:"daysRemaining,omitempty"`
	Overdue        bool       `json:"overdue"`
}

func toTaskResponse(t *models.Task) taskResponse {
	r := taskResponse{
		ID:          t.ID,
		UserID:      t.UserID,
		Title:       t.Title,
		Description: t.Description,
		DueDate:     t.DueDate,
		CategoryID:  t.CategoryID,
		Completed:   t.Completed,
		CreatedAt:   t.CreatedAt,
		CompletedAt: t.CompletedAt,
	}
	if t.DueDate != "" {
		if due, err := time.Parse("2006-01-02", t.DueDate); err == nil {
			now := time.Now().UTC()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			diff := int(due.Sub(today).Hours() / 24)
			r.DaysRemaining = &diff
			if !t.Completed && due.Before(today) {
				r.Overdue = true
			}
		}
	}
	return r
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"dueDate"`
	CategoryID  string `json:"categoryId"`
}

func (h *TaskHandler) validateCategory(userID, categoryID string) error {
	if categoryID == "" {
		return nil
	}
	c, err := h.Store.GetCategory(categoryID)
	if err != nil || c.UserID != userID {
		return errors.New("category not found")
	}
	return nil
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	categoryID := r.URL.Query().Get("categoryId")
	dueFrom := r.URL.Query().Get("dueFrom")
	dueTo := r.URL.Query().Get("dueTo")

	if err := validation.ValidateDate(dueFrom); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "dueFrom: "+err.Error())
		return
	}
	if err := validation.ValidateDate(dueTo); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "dueTo: "+err.Error())
		return
	}

	tasks := h.Store.ListTasks(userID)
	filtered := make([]*models.Task, 0, len(tasks))
	for _, t := range tasks {
		if categoryID != "" && t.CategoryID != categoryID {
			continue
		}
		if dueFrom != "" && (t.DueDate == "" || t.DueDate < dueFrom) {
			continue
		}
		if dueTo != "" && (t.DueDate == "" || t.DueDate > dueTo) {
			continue
		}
		filtered = append(filtered, t)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	out := make([]taskResponse, 0, len(filtered))
	for _, t := range filtered {
		out = append(out, toTaskResponse(t))
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	var req createTaskRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	title := validation.SanitizeString(req.Title)
	if title == "" {
		httputil.WriteError(w, http.StatusBadRequest, "title is required")
		return
	}
	if len(title) > 500 {
		httputil.WriteError(w, http.StatusBadRequest, "title too long")
		return
	}
	description := validation.SanitizeString(req.Description)
	dueDate := validation.SanitizeString(req.DueDate)
	if err := validation.ValidateDate(dueDate); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	categoryID := validation.SanitizeString(req.CategoryID)
	if err := h.validateCategory(userID, categoryID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	t := &models.Task{
		ID:          uuid.NewString(),
		UserID:      userID,
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		CategoryID:  categoryID,
		Completed:   false,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.Store.CreateTask(t); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toTaskResponse(t))
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	DueDate     *string `json:"dueDate"`
	CategoryID  *string `json:"categoryId"`
	Completed   *bool   `json:"completed"`
}

func (h *TaskHandler) getOwnedTask(w http.ResponseWriter, r *http.Request, id string) (*models.Task, bool) {
	userID := middleware.UserID(r)
	t, err := h.Store.GetTask(id)
	if err != nil || t.UserID != userID {
		httputil.WriteError(w, http.StatusNotFound, "task not found")
		return nil, false
	}
	return t, true
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	t, ok := h.getOwnedTask(w, r, id)
	if !ok {
		return
	}
	var req updateTaskRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated := *t
	if req.Title != nil {
		title := validation.SanitizeString(*req.Title)
		if title == "" {
			httputil.WriteError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		if len(title) > 500 {
			httputil.WriteError(w, http.StatusBadRequest, "title too long")
			return
		}
		updated.Title = title
	}
	if req.Description != nil {
		updated.Description = validation.SanitizeString(*req.Description)
	}
	if req.DueDate != nil {
		d := validation.SanitizeString(*req.DueDate)
		if err := validation.ValidateDate(d); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		updated.DueDate = d
	}
	if req.CategoryID != nil {
		c := validation.SanitizeString(*req.CategoryID)
		if err := h.validateCategory(t.UserID, c); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		updated.CategoryID = c
	}
	if req.Completed != nil {
		updated.Completed = *req.Completed
		if *req.Completed && updated.CompletedAt == nil {
			now := time.Now().UTC()
			updated.CompletedAt = &now
		}
		if !*req.Completed {
			updated.CompletedAt = nil
		}
	}
	if err := h.Store.UpdateTask(&updated); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toTaskResponse(&updated))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := h.getOwnedTask(w, r, id); !ok {
		return
	}
	if err := h.Store.DeleteTask(id); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request, id string) {
	t, ok := h.getOwnedTask(w, r, id)
	if !ok {
		return
	}
	updated := *t
	if !updated.Completed {
		updated.Completed = true
		now := time.Now().UTC()
		updated.CompletedAt = &now
	}
	if err := h.Store.UpdateTask(&updated); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to complete task")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toTaskResponse(&updated))
}

func (h *TaskHandler) Incomplete(w http.ResponseWriter, r *http.Request, id string) {
	t, ok := h.getOwnedTask(w, r, id)
	if !ok {
		return
	}
	updated := *t
	updated.Completed = false
	updated.CompletedAt = nil
	if err := h.Store.UpdateTask(&updated); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to uncomplete task")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toTaskResponse(&updated))
}
