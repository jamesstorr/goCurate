package models

// Erato Interfaces

// Create the Collector interface
type Collector interface {
	CatalogContents() error
	AllContentRefs() []interface{}
	DownloadContentData(interface{}) (*[]byte, error)
	// TODO - move to
	// AllContentRefs() []ContentRef
	// DownloadContentData(ContentRef) (*[]byte, error)
}

// ContentRef - Interface for the Data that is sourced
type ContentRef interface {
	GetUniqueID() string
	GetName() string
	GetFileName() string
	GetLocation() string
	GetTypeName() string
	GetPath() string
	GetPathHash() string
	GetParentLocation() string
}

type ContentPreparer interface {
	// Prepare(docData *[]byte, c *Config) ([]string, error)
	Prepare(docData *[]byte) ([]string, error)
}

type ContentAnalyser interface {
	NewContentAnalysis(EratoID string, content []string) ContentAnalysis
	AnalyserDisabled() bool
	// AnalyserType() string
	// AnalyserDebug() bool
}

type ContentAnalysis interface {
	AnalyseContent() error
	AnalysisResultCount() int
	AnalysisErrorCount() int
	AnalysisResultError(i int) error
	AnalysisResultData(i int) interface{}
}
