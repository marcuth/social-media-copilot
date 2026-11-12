package services

import "github.com/marcuth/social-media-copilot/internal/catalog"

type Catalog struct {
	Platforms []catalog.Platform `json:"platforms"`
	Formats   []catalog.Option   `json:"formats"`
	Statuses  []catalog.Option   `json:"statuses"`
}

type CatalogService struct{}

func (CatalogService) GetCatalog() Catalog {
	return Catalog{
		Platforms: catalog.Platforms,
		Formats:   catalog.Formats,
		Statuses:  catalog.Statuses,
	}
}
