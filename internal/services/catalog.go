package services

import "github.com/marcuth/social-media-copilot/internal/catalog"

// Catalog is everything the frontend needs to render pickers and labels.
type Catalog struct {
	Platforms []catalog.Platform `json:"platforms"`
	Formats   []catalog.Option   `json:"formats"`
	Statuses  []catalog.Option   `json:"statuses"`
}

// CatalogService exposes the fixed vocabularies.
type CatalogService struct{}

// GetCatalog returns platforms, formats and statuses.
func (CatalogService) GetCatalog() Catalog {
	return Catalog{
		Platforms: catalog.Platforms,
		Formats:   catalog.Formats,
		Statuses:  catalog.Statuses,
	}
}
