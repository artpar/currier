package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/artpar/currier/internal/history"
)

// registerResources registers all MCP resources
func (s *Server) registerResources() {
	s.registerCollectionsResource()
	s.registerHistoryResource()
}

func (s *Server) registerCollectionsResource() {
	s.resources["collections://list"] = &resourceDef{
		resource: Resource{
			URI:         "collections://list",
			Name:        "Collections List",
			Description: "List of all API collections",
			MimeType:    "application/json",
		},
		handler: func() (*ResourceReadResult, error) {
			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]map[string]any, 0, len(collections))
			for _, c := range collections {
				result = append(result, map[string]any{
					"name":          c.Name,
					"id":            c.ID,
					"description":   c.Description,
					"request_count": c.RequestCount,
				})
			}

			data, err := json.MarshalIndent(map[string]any{"collections": result}, "", "  ")
			if err != nil {
				return nil, err
			}

			return &ResourceReadResult{
				Contents: []ResourceContent{
					{
						URI:      "collections://list",
						MimeType: "application/json",
						Text:     string(data),
					},
				},
			}, nil
		},
	}
}

func (s *Server) registerHistoryResource() {
	s.resources["history://recent"] = &resourceDef{
		resource: Resource{
			URI:         "history://recent",
			Name:        "Recent History",
			Description: "Recent API request history",
			MimeType:    "application/json",
		},
		handler: func() (*ResourceReadResult, error) {
			ctx := context.Background()
			entries, err := s.history.List(ctx, history.QueryOptions{Limit: 50})
			if err != nil {
				return nil, err
			}

			result := make([]map[string]any, 0, len(entries))
			for _, e := range entries {
				result = append(result, map[string]any{
					"id":          e.ID,
					"method":      e.RequestMethod,
					"url":         e.RequestURL,
					"status":      e.ResponseStatus,
					"duration_ms": e.ResponseTime,
					"timestamp":   e.Timestamp.Format(time.RFC3339),
				})
			}

			data, err := json.MarshalIndent(map[string]any{"history": result}, "", "  ")
			if err != nil {
				return nil, err
			}

			return &ResourceReadResult{
				Contents: []ResourceContent{
					{
						URI:      "history://recent",
						MimeType: "application/json",
						Text:     string(data),
					},
				},
			}, nil
		},
	}
}
