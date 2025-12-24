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

// CollectionMeta contains metadata for listing collections.
type CollectionMeta struct {
	ID           string
	Name         string
	Description  string
	Path         string
	RequestCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CollectionStore manages collection persistence to the filesystem.
type CollectionStore struct {
	basePath string
}

// NewCollectionStore creates a new filesystem-based collection store.
func NewCollectionStore(basePath string) (*CollectionStore, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create collections directory: %w", err)
	}

	return &CollectionStore{
		basePath: basePath,
	}, nil
}

// Save persists a collection to disk.
func (s *CollectionStore) Save(ctx context.Context, c *core.Collection) error {
	data := s.toStorageFormat(c)

	content, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}

	path := s.collectionPath(c.ID())
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write collection file: %w", err)
	}

	return nil
}

// Get retrieves a collection by ID.
func (s *CollectionStore) Get(ctx context.Context, id string) (*core.Collection, error) {
	path := s.collectionPath(id)
	return s.loadFromPath(path)
}

// GetByPath retrieves a collection by file path.
func (s *CollectionStore) GetByPath(ctx context.Context, path string) (*core.Collection, error) {
	return s.loadFromPath(path)
}

// List returns all collections.
func (s *CollectionStore) List(ctx context.Context) ([]CollectionMeta, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read collections directory: %w", err)
	}

	var collections []CollectionMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(s.basePath, entry.Name())
		c, err := s.loadFromPath(path)
		if err != nil {
			continue // Skip invalid files
		}

		collections = append(collections, CollectionMeta{
			ID:           c.ID(),
			Name:         c.Name(),
			Description:  c.Description(),
			Path:         path,
			RequestCount: s.countRequests(c),
			CreatedAt:    c.CreatedAt(),
			UpdatedAt:    c.UpdatedAt(),
		})
	}

	return collections, nil
}

// Delete removes a collection.
func (s *CollectionStore) Delete(ctx context.Context, id string) error {
	path := s.collectionPath(id)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("collection not found: %s", id)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

// Search finds collections matching the query.
func (s *CollectionStore) Search(ctx context.Context, query string) ([]CollectionMeta, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []CollectionMeta

	for _, meta := range all {
		if strings.Contains(strings.ToLower(meta.Name), query) ||
			strings.Contains(strings.ToLower(meta.Description), query) {
			results = append(results, meta)
		}
	}

	return results, nil
}

// Internal helpers

func (s *CollectionStore) collectionPath(id string) string {
	return filepath.Join(s.basePath, id+".yaml")
}

func (s *CollectionStore) loadFromPath(path string) (*core.Collection, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection file: %w", err)
	}

	var data collectionData
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection: %w", err)
	}

	return s.fromStorageFormat(&data), nil
}

func (s *CollectionStore) countRequests(c *core.Collection) int {
	count := len(c.Requests())
	for _, f := range c.Folders() {
		count += s.countFolderRequests(f)
	}
	return count
}

func (s *CollectionStore) countFolderRequests(f *core.Folder) int {
	count := len(f.Requests())
	for _, sf := range f.Folders() {
		count += s.countFolderRequests(sf)
	}
	return count
}

// Storage format types

type collectionData struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Variables   map[string]string `yaml:"variables,omitempty"`
	Auth        authData          `yaml:"auth,omitempty"`
	PreScript   string            `yaml:"pre_script,omitempty"`
	PostScript  string            `yaml:"post_script,omitempty"`
	Folders     []folderData      `yaml:"folders,omitempty"`
	Requests    []requestData     `yaml:"requests,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at"`
	UpdatedAt   time.Time         `yaml:"updated_at"`
}

type folderData struct {
	ID          string        `yaml:"id"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description,omitempty"`
	Folders     []folderData  `yaml:"folders,omitempty"`
	Requests    []requestData `yaml:"requests,omitempty"`
}

type requestData struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Method      string            `yaml:"method"`
	URL         string            `yaml:"url"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	BodyType    string            `yaml:"body_type,omitempty"`
	BodyContent string            `yaml:"body_content,omitempty"`
	Auth        *authData         `yaml:"auth,omitempty"`
	PreScript   string            `yaml:"pre_script,omitempty"`
	PostScript  string            `yaml:"post_script,omitempty"`
}

type authData struct {
	Type     string `yaml:"type,omitempty"`
	Token    string `yaml:"token,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Key      string `yaml:"key,omitempty"`
	Value    string `yaml:"value,omitempty"`
	In       string `yaml:"in,omitempty"`
}

// Conversion functions

func (s *CollectionStore) toStorageFormat(c *core.Collection) *collectionData {
	data := &collectionData{
		ID:          c.ID(),
		Name:        c.Name(),
		Description: c.Description(),
		Version:     c.Version(),
		Variables:   c.Variables(),
		Auth:        toAuthData(c.Auth()),
		PreScript:   c.PreScript(),
		PostScript:  c.PostScript(),
		CreatedAt:   c.CreatedAt(),
		UpdatedAt:   c.UpdatedAt(),
	}

	for _, f := range c.Folders() {
		data.Folders = append(data.Folders, s.toFolderData(f))
	}

	for _, r := range c.Requests() {
		data.Requests = append(data.Requests, s.toRequestData(r))
	}

	return data
}

func (s *CollectionStore) toFolderData(f *core.Folder) folderData {
	data := folderData{
		ID:          f.ID(),
		Name:        f.Name(),
		Description: f.Description(),
	}

	for _, sf := range f.Folders() {
		data.Folders = append(data.Folders, s.toFolderData(sf))
	}

	for _, r := range f.Requests() {
		data.Requests = append(data.Requests, s.toRequestData(r))
	}

	return data
}

func (s *CollectionStore) toRequestData(r *core.RequestDefinition) requestData {
	return requestData{
		ID:          r.ID(),
		Name:        r.Name(),
		Description: r.Description(),
		Method:      r.Method(),
		URL:         r.URL(),
		Headers:     r.Headers(),
		BodyType:    r.BodyType(),
		BodyContent: r.BodyContent(),
		PreScript:   r.PreScript(),
		PostScript:  r.PostScript(),
	}
}

func toAuthData(a core.AuthConfig) authData {
	return authData{
		Type:     a.Type,
		Token:    a.Token,
		Username: a.Username,
		Password: a.Password,
		Key:      a.Key,
		Value:    a.Value,
		In:       a.In,
	}
}

func (s *CollectionStore) fromStorageFormat(data *collectionData) *core.Collection {
	c := core.NewCollectionWithID(data.ID, data.Name)
	c.SetDescription(data.Description)
	c.SetVersion(data.Version)
	c.SetAuth(fromAuthData(data.Auth))
	c.SetPreScript(data.PreScript)
	c.SetPostScript(data.PostScript)
	c.SetTimestamps(data.CreatedAt, data.UpdatedAt)

	for k, v := range data.Variables {
		c.SetVariable(k, v)
	}

	for _, fd := range data.Folders {
		f := s.fromFolderData(&fd)
		c.AddExistingFolder(f)
	}

	for _, rd := range data.Requests {
		r := s.fromRequestData(&rd)
		c.AddRequest(r)
	}

	return c
}

func (s *CollectionStore) fromFolderData(data *folderData) *core.Folder {
	f := core.NewFolderWithID(data.ID, data.Name)
	f.SetDescription(data.Description)

	for _, fd := range data.Folders {
		sf := s.fromFolderData(&fd)
		f.AddExistingFolder(sf)
	}

	for _, rd := range data.Requests {
		r := s.fromRequestData(&rd)
		f.AddRequest(r)
	}

	return f
}

func (s *CollectionStore) fromRequestData(data *requestData) *core.RequestDefinition {
	r := core.NewRequestDefinitionWithID(data.ID, data.Name, data.Method, data.URL)
	r.SetDescription(data.Description)
	r.SetPreScript(data.PreScript)
	r.SetPostScript(data.PostScript)

	for k, v := range data.Headers {
		r.SetHeader(k, v)
	}

	if data.BodyContent != "" {
		r.SetBodyRaw(data.BodyContent, data.BodyType)
	}

	return r
}

func fromAuthData(data authData) core.AuthConfig {
	return core.AuthConfig{
		Type:     data.Type,
		Token:    data.Token,
		Username: data.Username,
		Password: data.Password,
		Key:      data.Key,
		Value:    data.Value,
		In:       data.In,
	}
}
