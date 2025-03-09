package erato

import (
	"Erato/erato/analysers/openai"
	sharepoint "Erato/erato/collectors/sharepoint"
	website "Erato/erato/collectors/website"
	"Erato/erato/preparers/content"
	"time"

	"Erato/erato/models"
	"fmt"
	"log"
)

const (
	// OpenAIworkerDelay = 500
	// EratoWrokerDelay  = 500
	EratoWrokerDelay = 1000
)

// Erato - Main Struct for the Erato Application
type Erato struct {
	Conf               *Conf
	ContentCollections []Collection
	EratoCollectors    EratoCollectors
	EratoPreparer      content.Config
	EratoAnalysers     EratoAnalysers
	EratoCollections   Collection
}

// Analysers - Limited to 1-2-1 relationships
type EratoAnalysers struct {
	OpenAI *openai.OpenAI
}

type Collection struct {
	Name                 string
	ContentSource        ContentSource
	ContentCatalog       ContentCatalog
	ContentPreparer      content.Config
	ContentAnalyser      models.ContentAnalyser
	ContentCatalogsStats ContentCatalogAnalysisStats
	Conf                 *Conf
}

type Document struct {
	Analyser       models.ContentAnalyser
	Collector      models.Collector
	EratoContentID string
	SourceID       string
	Location       string
	ParentLocation string
	Name           string
	FileName       string
	Path           string
	PathHash       string
	Type           string
	FileExt        string
	ContentType    interface{}
	ContentRef     interface{}
	ContentSource  string
	OAIPrompt      string
	// ParagraphText   []string
	// Documents have to be text in some form
	NumTextChunks int
	TextChunks    []string
	// DocMetaData     []ParagraphMetaData
	DocMetaData     []interface{}
	TypeDocMetaData map[string][]ParagraphMetaData
	DocumentData    *[]byte
	Curated         bool
	AnalysisStats   DocumentAnalysisStats
	AnalysisErrors  []error
}

// NewErato - Setup everything from the config files
func NewErato2(conf *EratoConf) (*Erato, error) {
	var err error
	var e Erato

	return &e, err
}

// NewErato - Setup everything from the config files
func NewErato(env string) (*Erato, error) {
	var err error
	var spc *sharepoint.SharePointColector
	var web *website.WebsiteCollector
	// There is a seperation between the Config and
	// 		the Erato Object
	// 		the Collectors etc.

	// Get the Config
	c := NewConfig()

	// Setup the collectors
	// Todo - move to Collector setup function
	sc := c.SharePoint
	spc, err = sharepoint.NewCollector(&sc)
	if err != nil {
		log.Fatal(err)
	}

	wc := c.Website
	web, err = website.NewCollector(&wc)
	if err != nil {
		log.Fatal(err)
	}

	oai, _ := openai.NewOpenAI(&c.XX_OAI)

	e := Erato{
		Conf: c,
		EratoCollectors: EratoCollectors{
			Sharepoint: spc,
			Website:    web,
		},
		EratoAnalysers: EratoAnalysers{
			OpenAI: oai,
		},
		EratoPreparer: c.ContentPreparer,
	}

	return &e, err
}

// printAnalysisStats - Print the Analysis Stats
func printAnalysisStats(eratoStats ContentCatalogAnalysisStats, catalogName string) {

	fmt.Println("*____________________________________________________________________________________*")
	// if eratoStats.Errors > 0 || eratoStats.Warnings > 0 {
	fmt.Printf("Erato - ERRORS Content Catalog=%v\n"+
		"\tFound=%v\n"+
		"\tAnalysed=%v\n"+
		"\tSuccesses=%v\n"+
		"\tErrors=%v\n"+
		"\tWarnings=%v\n",
		catalogName,
		eratoStats.Found,
		eratoStats.Analysed,
		eratoStats.Successes,
		eratoStats.Errors,
		eratoStats.Warnings)

	// } else {
	// fmt.Printf("Erato - Success - Content Catalog=%v - Analysed=%v\n", catalogName, eratoStats.Analysed)
	// }
	fmt.Println("*____________________________________________________________________________________*")

}

func (collection *Collection) DumpCatalogFileNames() {

	// List the collected content in the content catalogues
	fmt.Println("*____________________________________________________________________________________*")
	fmt.Printf("ContentCatalog=%v\n", collection.Name)
	for _, doc := range collection.ContentCatalog {
		fmt.Printf("Erato - DEBUG - ContentCatalog:%v - FileName:%v\n", collection.Name, doc.FileName)
	}

	if len(collection.ContentCatalog) == 0 {
		fmt.Printf("Erato - DEBUG  - ContentCatalog:%v - Warning No Documents Found\n", collection.Name)
	}

}

func (doc *Document) ReportDocumentAnalysisStats() {
	var stats string
	fmt.Println("\n*--------------------------------------------------------------------------------*")

	fmt.Printf("Erato - Processing Statistics - %v\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Erato - Processing Statistics - Document:%v\n", doc.FileName)

	if doc.AnalysisStats.Errors > 0 || doc.AnalysisStats.Warnings > 0 {
		cnt := doc.AnalysisStats.Errors + doc.AnalysisStats.Warnings
		fmt.Printf("**ERRORS** Erato - Processing Statistics - Total %v Processing Problems Found\n", cnt)
	}

	stats = fmt.Sprintf("Document Analysis Stats - To Process:%v, Processed:%v, Success:%v, Errors:%v, Warnings:%v\n", doc.AnalysisStats.ToProcess, doc.AnalysisStats.Processed, doc.AnalysisStats.Success, doc.AnalysisStats.Errors, doc.AnalysisStats.Warnings)
	fmt.Printf("Erato - Processing Statistics - Document:%v\n", stats)

	fmt.Println("*--------------------------------------------------------------------------------*")

}
