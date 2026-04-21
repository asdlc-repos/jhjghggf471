package store

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/asdlc/task-api/internal/models"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate")
)

type Store interface {
	// Users
	CreateUser(u *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateUserPassword(id string, passwordHash []byte) error

	// Sessions
	CreateSession(s *models.Session) error
	GetSession(token string) (*models.Session, error)
	DeleteSession(token string) error

	// Login attempts
	GetLoginAttempt(email string) *models.LoginAttempt
	SetLoginAttempt(email string, a *models.LoginAttempt)
	ClearLoginAttempt(email string)

	// Password reset tokens
	CreateResetToken(t *models.PasswordResetToken) error
	GetResetToken(token string) (*models.PasswordResetToken, error)
	DeleteResetToken(token string)

	// Tasks
	CreateTask(t *models.Task) error
	GetTask(id string) (*models.Task, error)
	ListTasks(userID string) []*models.Task
	UpdateTask(t *models.Task) error
	DeleteTask(id string) error
	UnassignCategory(userID, categoryID string)

	// Categories
	CreateCategory(c *models.Category) error
	GetCategory(id string) (*models.Category, error)
	ListCategories(userID string) []*models.Category
	UpdateCategory(c *models.Category) error
	DeleteCategory(id string) error
}

type memoryStore struct {
	usersMu      sync.RWMutex
	users        map[string]*models.User
	usersByEmail map[string]string

	sessionsMu sync.RWMutex
	sessions   map[string]*models.Session

	attemptsMu sync.Mutex
	attempts   map[string]*models.LoginAttempt

	resetMu     sync.RWMutex
	resetTokens map[string]*models.PasswordResetToken

	tasksMu sync.RWMutex
	tasks   map[string]*models.Task

	categoriesMu sync.RWMutex
	categories   map[string]*models.Category
}

func NewMemoryStore() Store {
	return &memoryStore{
		users:        make(map[string]*models.User),
		usersByEmail: make(map[string]string),
		sessions:     make(map[string]*models.Session),
		attempts:     make(map[string]*models.LoginAttempt),
		resetTokens:  make(map[string]*models.PasswordResetToken),
		tasks:        make(map[string]*models.Task),
		categories:   make(map[string]*models.Category),
	}
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// --- Users ---

func (s *memoryStore) CreateUser(u *models.User) error {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()
	email := NormalizeEmail(u.Email)
	if _, ok := s.usersByEmail[email]; ok {
		return ErrDuplicate
	}
	u.Email = email
	s.users[u.ID] = u
	s.usersByEmail[email] = u.ID
	return nil
}

func (s *memoryStore) GetUserByEmail(email string) (*models.User, error) {
	email = NormalizeEmail(email)
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	id, ok := s.usersByEmail[email]
	if !ok {
		return nil, ErrNotFound
	}
	return s.users[id], nil
}

func (s *memoryStore) GetUserByID(id string) (*models.User, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *memoryStore) UpdateUserPassword(id string, passwordHash []byte) error {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return ErrNotFound
	}
	u.PasswordHash = passwordHash
	return nil
}

// --- Sessions ---

func (s *memoryStore) CreateSession(sess *models.Session) error {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[sess.Token] = sess
	return nil
}

func (s *memoryStore) GetSession(token string) (*models.Session, error) {
	s.sessionsMu.RLock()
	sess, ok := s.sessions[token]
	s.sessionsMu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if time.Now().After(sess.ExpiresAt) {
		s.sessionsMu.Lock()
		delete(s.sessions, token)
		s.sessionsMu.Unlock()
		return nil, ErrNotFound
	}
	return sess, nil
}

func (s *memoryStore) DeleteSession(token string) error {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, token)
	return nil
}

// --- Login attempts ---

func (s *memoryStore) GetLoginAttempt(email string) *models.LoginAttempt {
	email = NormalizeEmail(email)
	s.attemptsMu.Lock()
	defer s.attemptsMu.Unlock()
	a, ok := s.attempts[email]
	if !ok {
		return nil
	}
	cp := *a
	return &cp
}

func (s *memoryStore) SetLoginAttempt(email string, a *models.LoginAttempt) {
	email = NormalizeEmail(email)
	s.attemptsMu.Lock()
	defer s.attemptsMu.Unlock()
	s.attempts[email] = a
}

func (s *memoryStore) ClearLoginAttempt(email string) {
	email = NormalizeEmail(email)
	s.attemptsMu.Lock()
	defer s.attemptsMu.Unlock()
	delete(s.attempts, email)
}

// --- Password reset tokens ---

func (s *memoryStore) CreateResetToken(t *models.PasswordResetToken) error {
	s.resetMu.Lock()
	defer s.resetMu.Unlock()
	s.resetTokens[t.Token] = t
	return nil
}

func (s *memoryStore) GetResetToken(token string) (*models.PasswordResetToken, error) {
	s.resetMu.RLock()
	t, ok := s.resetTokens[token]
	s.resetMu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if time.Now().After(t.ExpiresAt) {
		s.resetMu.Lock()
		delete(s.resetTokens, token)
		s.resetMu.Unlock()
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *memoryStore) DeleteResetToken(token string) {
	s.resetMu.Lock()
	defer s.resetMu.Unlock()
	delete(s.resetTokens, token)
}

// --- Tasks ---

func (s *memoryStore) CreateTask(t *models.Task) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	s.tasks[t.ID] = t
	return nil
}

func (s *memoryStore) GetTask(id string) (*models.Task, error) {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *memoryStore) ListTasks(userID string) []*models.Task {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()
	out := make([]*models.Task, 0)
	for _, t := range s.tasks {
		if t.UserID == userID {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out
}

func (s *memoryStore) UpdateTask(t *models.Task) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	if _, ok := s.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	s.tasks[t.ID] = t
	return nil
}

func (s *memoryStore) DeleteTask(id string) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}

func (s *memoryStore) UnassignCategory(userID, categoryID string) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	for _, t := range s.tasks {
		if t.UserID == userID && t.CategoryID == categoryID {
			t.CategoryID = ""
		}
	}
}

// --- Categories ---

func (s *memoryStore) CreateCategory(c *models.Category) error {
	s.categoriesMu.Lock()
	defer s.categoriesMu.Unlock()
	lower := strings.ToLower(strings.TrimSpace(c.Name))
	for _, existing := range s.categories {
		if existing.UserID == c.UserID && strings.ToLower(existing.Name) == lower {
			return ErrDuplicate
		}
	}
	s.categories[c.ID] = c
	return nil
}

func (s *memoryStore) GetCategory(id string) (*models.Category, error) {
	s.categoriesMu.RLock()
	defer s.categoriesMu.RUnlock()
	c, ok := s.categories[id]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *memoryStore) ListCategories(userID string) []*models.Category {
	s.categoriesMu.RLock()
	defer s.categoriesMu.RUnlock()
	out := make([]*models.Category, 0)
	for _, c := range s.categories {
		if c.UserID == userID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out
}

func (s *memoryStore) UpdateCategory(c *models.Category) error {
	s.categoriesMu.Lock()
	defer s.categoriesMu.Unlock()
	existing, ok := s.categories[c.ID]
	if !ok {
		return ErrNotFound
	}
	lower := strings.ToLower(strings.TrimSpace(c.Name))
	for id, other := range s.categories {
		if id == c.ID {
			continue
		}
		if other.UserID == c.UserID && strings.ToLower(other.Name) == lower {
			return ErrDuplicate
		}
	}
	existing.Name = c.Name
	return nil
}

func (s *memoryStore) DeleteCategory(id string) error {
	s.categoriesMu.Lock()
	defer s.categoriesMu.Unlock()
	if _, ok := s.categories[id]; !ok {
		return ErrNotFound
	}
	delete(s.categories, id)
	return nil
}
