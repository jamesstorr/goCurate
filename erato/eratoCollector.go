package erato

import (
	sharepoint "Erato/erato/collectors/sharepoint"
	website "Erato/erato/collectors/website"
	"Erato/erato/models"
)

func (coll *Collection) ContentCollector() models.Collector {
	return coll.ContentSource.Collector
}

// Collectors - Limited to 1-2-1 relationships
type EratoCollectors struct {
	Sharepoint *sharepoint.SharePointColector
	Website    *website.WebsiteCollector
}

// Where and how to get the data
type ContentSource struct {
	Name      string
	Collector models.Collector
}
