package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/artpar/currier/internal/core"
	"gopkg.in/yaml.v3"
)

// EnvironmentMeta contains metadata for listing environments.
type EnvironmentMeta struct {
	ID        string
	Name      string
	IsActive  bool
	VarCount  int
	UpdatedAt time.Time
}

// EnvironmentStore manages environment persistence to the filesystem.
type EnvironmentStore struct {
	basePath string
}

// NewEnvironmentStore creates a new filesystem-based environment store.
func NewEnvironmentStore(basePath string) (*EnvironmentStore, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create environments directory: %w", err)
	}

	return &EnvironmentStore{
		basePath: basePath,
	}, nil
}

// Save persists an environment to disk.
func (s *EnvironmentStore) Save(ctx context.Context, env *core.Environment) error {
	data := s.toStorageFormat(env)

	content, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal environment: %w", err)
	}

	path := s.environmentPath(env.ID())
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}

	return nil
}

// Get retrieves an environment by ID.
func (s *EnvironmentStore) Get(ctx context.Context, id string) (*core.Environment, error) {
	path := s.environmentPath(id)
	return s.loadFromPath(path)
}

// GetByName retrieves an environment by name.
func (s *EnvironmentStore) GetByName(ctx context.Context, name string) (*core.Environment, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environments directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(s.basePath, entry.Name())
		env, err := s.loadFromPath(path)
		if err != nil {
			continue
		}

		if env.Name() == name {
			return env, nil
		}
	}

	return nil, fmt.Errorf("environment not found: %s", name)
}

// List returns all environments.
func (s *EnvironmentStore) List(ctx context.Context) ([]EnvironmentMeta, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environments directory: %w", err)
	}

	var environments []EnvironmentMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(s.basePath, entry.Name())
		env, err := s.loadFromPath(path)
		if err != nil {
			continue
		}

		environments = append(environments, EnvironmentMeta{
			ID:        env.ID(),
			Name:      env.Name(),
			IsActive:  env.IsActive(),
			VarCount:  len(env.Variables()),
			UpdatedAt: env.UpdatedAt(),
		})
	}

	return environments, nil
}

// Delete removes an environment.
func (s *EnvironmentStore) Delete(ctx context.Context, id string) error {
	path := s.environmentPath(id)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("environment not found: %s", id)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	return nil
}

// GetActive returns the currently active environment.
func (s *EnvironmentStore) GetActive(ctx context.Context) (*core.Environment, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environments directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(s.basePath, entry.Name())
		env, err := s.loadFromPath(path)
		if err != nil {
			continue
		}

		if env.IsActive() {
			return env, nil
		}
	}

	return nil, fmt.Errorf("no active environment")
}

// SetActive sets the active environment (deactivates all others).
func (s *EnvironmentStore) SetActive(ctx context.Context, id string) error {
	// First verify the target environment exists
	targetPath := s.environmentPath(id)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("environment not found: %s", id)
	}

	// Deactivate all environments
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to read environments directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(s.basePath, entry.Name())
		env, err := s.loadFromPath(path)
		if err != nil {
			continue
		}

		// Set active status
		isTarget := env.ID() == id
		if env.IsActive() != isTarget {
			env.SetActive(isTarget)
			if err := s.Save(ctx, env); err != nil {
				return fmt.Errorf("failed to update environment: %w", err)
			}
		}
	}

	return nil
}

// Internal helpers

func (s *EnvironmentStore) environmentPath(id string) string {
	return filepath.Join(s.basePath, id+".yaml")
}

func (s *EnvironmentStore) loadFromPath(path string) (*core.Environment, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %w", err)
	}

	var data environmentData
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}

	return s.fromStorageFormat(&data), nil
}

// Storage format types

type environmentData struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Variables   map[string]string `yaml:"variables,omitempty"`
	Secrets     map[string]string `yaml:"secrets,omitempty"`
	IsActive    bool              `yaml:"is_active"`
	IsGlobal    bool              `yaml:"is_global"`
	CreatedAt   time.Time         `yaml:"created_at"`
	UpdatedAt   time.Time         `yaml:"updated_at"`
}

// Conversion functions

func (s *EnvironmentStore) toStorageFormat(env *core.Environment) *environmentData {
	return &environmentData{
		ID:          env.ID(),
		Name:        env.Name(),
		Description: env.Description(),
		Variables:   env.Variables(),
		Secrets:     s.getSecrets(env),
		IsActive:    env.IsActive(),
		IsGlobal:    env.IsGlobal(),
		CreatedAt:   env.CreatedAt(),
		UpdatedAt:   env.UpdatedAt(),
	}
}

func (s *EnvironmentStore) getSecrets(env *core.Environment) map[string]string {
	secrets := make(map[string]string)
	for _, name := range env.SecretNames() {
		secrets[name] = env.GetSecret(name)
	}
	return secrets
}

func (s *EnvironmentStore) fromStorageFormat(data *environmentData) *core.Environment {
	env := core.NewEnvironmentWithID(data.ID, data.Name)
	env.SetDescription(data.Description)
	env.SetActive(data.IsActive)
	env.SetGlobal(data.IsGlobal)
	env.SetTimestamps(data.CreatedAt, data.UpdatedAt)

	for k, v := range data.Variables {
		env.SetVariable(k, v)
	}

	for k, v := range data.Secrets {
		env.SetSecret(k, v)
	}

	return env
}
