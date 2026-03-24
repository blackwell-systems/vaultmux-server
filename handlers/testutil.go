package handlers

import (
	"context"
	"sync"

	"github.com/blackwell-systems/vaultmux"
)

// MockBackend implements vaultmux.Backend interface for testing.
// Uses in-memory map for secret storage with error injection capabilities.
type MockBackend struct {
	items map[string]*vaultmux.Item
	mu    sync.RWMutex

	// Error injection fields for testing error paths
	listErr   error
	getErr    error
	createErr error
	updateErr error
	deleteErr error
	existsErr error
}

// NewMockBackend creates a new mock backend for testing.
func NewMockBackend() *MockBackend {
	return &MockBackend{
		items: make(map[string]*vaultmux.Item),
	}
}

// Name returns the backend name.
func (m *MockBackend) Name() string {
	return "mock"
}

// Init is a no-op for mock backend.
func (m *MockBackend) Init(ctx context.Context) error {
	return nil
}

// Close is a no-op for mock backend.
func (m *MockBackend) Close() error {
	return nil
}

// IsAuthenticated always returns true for mock backend.
func (m *MockBackend) IsAuthenticated(ctx context.Context) bool {
	return true
}

// Authenticate returns a nil session (adequate for testing).
func (m *MockBackend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	return nil, nil
}

// Sync is a no-op for mock backend.
func (m *MockBackend) Sync(ctx context.Context, session vaultmux.Session) error {
	return nil
}

// GetItem retrieves an item from the in-memory store.
func (m *MockBackend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[name]
	if !ok {
		return nil, vaultmux.ErrNotFound
	}

	// Return a copy to avoid race conditions
	itemCopy := *item
	return &itemCopy, nil
}

// GetNotes retrieves just the notes field.
func (m *MockBackend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[name]
	if !ok {
		return "", vaultmux.ErrNotFound
	}

	return item.Notes, nil
}

// ItemExists checks if an item exists.
func (m *MockBackend) ItemExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.items[name]
	return ok, nil
}

// ListItems returns all items.
func (m *MockBackend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]*vaultmux.Item, 0, len(m.items))
	for _, item := range m.items {
		itemCopy := *item
		items = append(items, &itemCopy)
	}

	return items, nil
}

// CreateItem creates a new item in the in-memory store.
func (m *MockBackend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	if m.createErr != nil {
		return m.createErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[name] = &vaultmux.Item{
		ID:    name,
		Name:  name,
		Notes: content,
	}

	return nil
}

// UpdateItem updates an existing item in the in-memory store.
func (m *MockBackend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	if m.updateErr != nil {
		return m.updateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[name]
	if !ok {
		return vaultmux.ErrNotFound
	}

	item.Notes = content
	return nil
}

// DeleteItem deletes an item from the in-memory store.
func (m *MockBackend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.items[name]; !ok {
		return vaultmux.ErrNotFound
	}

	delete(m.items, name)
	return nil
}

// ListLocations returns an empty list (not used in handlers).
func (m *MockBackend) ListLocations(ctx context.Context, session vaultmux.Session) ([]string, error) {
	return []string{}, nil
}

// LocationExists returns false (not used in handlers).
func (m *MockBackend) LocationExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
	return false, nil
}

// CreateLocation is a no-op (not used in handlers).
func (m *MockBackend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
	return nil
}

// ListItemsInLocation returns items in a specific location (not used in handlers).
func (m *MockBackend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
	return []*vaultmux.Item{}, nil
}

// Error injection methods for testing error paths

// SetListError injects an error for ListItems calls.
func (m *MockBackend) SetListError(err error) {
	m.listErr = err
}

// SetGetError injects an error for GetNotes and GetItem calls.
func (m *MockBackend) SetGetError(err error) {
	m.getErr = err
}

// SetCreateError injects an error for CreateItem calls.
func (m *MockBackend) SetCreateError(err error) {
	m.createErr = err
}

// SetUpdateError injects an error for UpdateItem calls.
func (m *MockBackend) SetUpdateError(err error) {
	m.updateErr = err
}

// SetDeleteError injects an error for DeleteItem calls.
func (m *MockBackend) SetDeleteError(err error) {
	m.deleteErr = err
}

// SetExistsError injects an error for ItemExists calls.
func (m *MockBackend) SetExistsError(err error) {
	m.existsErr = err
}
