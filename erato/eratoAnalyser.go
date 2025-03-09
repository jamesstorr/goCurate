package erato

import (
	"Erato/erato/models"
	"Erato/erato/utils"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MetatData for a ParaGraph
type ParagraphMetaData struct {
	ParagraphNum      int
	ParagraphType     string   `json:"Paragraph Type"`
	ParagraphText     string   `json:"-"`
	ParagraphSummary  string   `json:"Paragraph Summary"`
	ParagraphQuestion string   `json:"Paragraph Question"`
	ClientName        []string `json:"Client Names"`
	Projects          []string `json:"Projects"`
	Technologies      []string `json:"Technologies"`
	Methods           []string `json:"Methods"`
	PeopleNames       []string `json:"People Names"`
	OrganisationNames []string `json:"Organisation Names"`
	BusinessUnits     []string `json:"Business"`
	// To be implemented
	// QuantativeInformation []string `json:"Quantative Information"`
	// ResultOutcome         string   `json:"Result or Outcome"`
}

// DocAnalysisResult - Doc Analysis Struct for worker results
type DocAnalysisResult struct {
	Order    int
	Document *Document
	Err      error
}

// DocumentAnalysisStats - Stats for the Document Analysis
type DocumentAnalysisStats struct {
	ToProcess int
	Processed int
	Success   int
	Errors    int
	Warnings  int
}

// ContentCatalogAnalysisStats - for Tracking the Analysis statistics
type ContentCatalogAnalysisStats struct {
	Found     int
	Analysed  int
	Successes int
	Errors    int
	Warnings  int
}

// AnalyseContentCatalog - Iterate through the Content Catalogue and Lanuch the Document Analysis
// This controls the parralel processing of the analysis calling LanuchAnalyseDocument to do the real work
func (collection *Collection) AnalyseContentCatalog() error {
	var err error
	var wg2 sync.WaitGroup
	debug := collection.Conf.Debug
	analysisWorkers := collection.Conf.EratoAnalysisWorkers

	ContCat := collection.ContentCatalog
	catalogName := collection.Name

	fmt.Println("*____________________________________________________________________________________*")
	fmt.Printf("Analysing All documents in Content Catalog=%v\n", catalogName)

	// Set the worker numbers
	wrkNum := 0
	procCount := 0

	// Iterate through the documents in the content catalog
	for i := range ContCat {
		// Get the pointer from the content catalog list
		doc := &ContCat[i]

		// Check to see if the number of workers has been reached
		if wrkNum == analysisWorkers {
			// Wait for the workers to finish
			wg2.Wait()
			wrkNum = 0
		}

		// Analyse the document
		wg2.Add(1)
		wrkNum++
		procCount++
		// TODO - stats.Processed++

		// Launch the worker to analyse the document funciton wrapped to ensure it is given the correct data
		go func(doc *Document, i int) {
			if debug {
				fmt.Printf("Erato - DEBUG - LaunchAnalyseDocument:%v - %v\n", i, doc.Name)
			}

			// TOOD - simplify by adding the info to the Document object
			// Launch the worker to analyse the document
			doc.AnalyseDocument(i, &wg2, collection)

		}(doc, i)

	}

	// Final wait for the workers to finish from the last batch of workers
	wg2.Wait()

	if debug {
		fmt.Println("Erato - DEBUG - Launched All Document Analysis Workers")
	}

	// TODO - Move to the document Processing loop if possible - with a function.
	// Work through the statistics of the Document Analysis
	fmt.Println("\nErato - Processing Analysis Results")
	eratoStats := collection.ContentCatalogsStats
	eratoStats.Found = len(collection.ContentCatalog)

	for i := range ContCat {
		doc := &ContCat[i]

		// Check Processing has happened
		if doc.AnalysisStats.Processed > 0 {
			eratoStats.Analysed++
		}

		// check the error status of the analysed document
		if doc.AnalysisStats.Errors == 0 && doc.AnalysisStats.Warnings == 0 {
			eratoStats.Successes++
			if debug {
				fmt.Printf("Erato - Error in analysing Document:%v - Error:%v\n", doc.FileName, doc.AnalysisErrors)
			}
		}

		// check the error status of the analysed document
		if doc.AnalysisStats.Errors > 0 {
			eratoStats.Errors++
			if debug {
				fmt.Printf("Erato - Error in analysing Document:%v - Error:%v\n", doc.FileName, doc.AnalysisErrors)
			}
		}

		// TODO Implement Warnings
		if doc.AnalysisStats.Warnings > 0 {
			eratoStats.Warnings++
			if debug {
				fmt.Printf("Erato - Warnings in analysing Document:%v - Warn:\n", doc.FileName)
			}
		}

	}

	// Print the final stats
	printAnalysisStats(eratoStats, catalogName)

	return err

}

// Wrapper to launch AnalyseDocument as a worker from AnalyseContentCatalog
// The process is as follows:
// 1. Download the document data
// 2. Prepare the document data
// 3. Analyse the document data
// 4. Store the document data
// 5. Update the document stats
func (doc *Document) AnalyseDocument(i int, wg *sync.WaitGroup, collection *Collection) error {

	defer wg.Done()
	var err error
	debug := collection.Conf.Debug

	// Get the Content Source Config
	contentCollector := doc.Collector
	if contentCollector == nil {
		log.Fatal("AnalyseDocument - Error - ContentCollector is nil")
	}

	// Sleeper to slow down the workers
	// TODO - Sleep time to config
	// time.Sleep(collection.Conf.???? * time.Millisecond)
	time.Sleep(500 * time.Millisecond)

	// Progress of Analysis
	if debug {
		fmt.Printf("\t\tLaunchAnalyseDocument - DEBUG - Analysing Document - %v - FileName:%v\n", i, doc.FileName)
	} else {
		fmt.Printf("\n(%v)-", i)
	}

	// Reference to the content data - required for downloading the document with the reference from the collector
	cRef := doc.ContentRef
	if cRef == nil {
		return errors.New("LaunchAnalyseDocument - Error - ContentRef is nil")
	}

	// Download the data for the conent and update the Document temporarly
	doc.DocumentData, err = contentCollector.DownloadContentData(cRef)
	if err != nil {
		fmt.Printf("\nLaunchAnalyseDocument - Error downloading FileName:%v with error:%v", doc.FileName, err)
		log.Println(err)
		doc.AnalysisStats.Errors++

		// Bail if you can't download the content
		doc.AnalysisErrors = append(doc.AnalysisErrors, err)
		// return
	}

	if debug {
		fmt.Printf("\nLaunchAnalyseDocument - Downloaded:%v - size:%v\n", filepath.Base(doc.FileName), len((*doc.DocumentData)))
	}

	// Prepare the Content for the Analyser driven by
	// the content type using the models.ContentPreparer
	// Test that the interface{} implements the ContentPreparer
	if contentTyper, ok := doc.ContentType.(models.ContentPreparer); ok {
		doc.TextChunks, err = contentTyper.Prepare(doc.DocumentData)
		if err != nil {
			return err
		}
	} else {
		// Something has gone very wrong if the type is not set
		// Exit here as there is no point analysing the document if the content can't be prepared
		log.Fatal(fmt.Errorf("analyseDocument - unsupported content type: %T", doc.ContentType))
	}

	// run the document analysis
	err = doc.contentAnalyserLauncher(debug)
	if err != nil {
		log.Printf("\tLaunchAnalyseDocument - %v - Error in analysing FileName:%v - Error:%v\n", i, doc.FileName, err)
		log.Println(err)
		// Update the Error Count of the Document
		doc.AnalysisStats.Errors++
		return err
	}

	// Store the document analysis
	if debug {
		fmt.Printf("\tLaunchAnalyseDocument - %v -  Storing the FileName:%v\n", i, doc.FileName)
	}

	err = doc.StoreDocumentAnalysis(collection.Conf)
	if err != nil {
		log.Printf("\tLaunchAnalyseDocument - %v - Error in storing FileName:%v - Error:%v\n", i, doc.FileName, err)
		// }

		// Update the Error Status of the Document
		doc.AnalysisErrors = append(doc.AnalysisErrors, err)
		doc.AnalysisStats.Errors++
	}

	return err

}

// StoreContentCatalog
func (collection *Collection) StoreContentCatalog() error {

	var err error
	ContCat := collection.ContentCatalog
	debug := collection.Conf.Debug

	// Iterate through the documents in the content catalog
	for i := range ContCat {
		doc := &ContCat[i]

		// func (doc *Document) AnalyseDocument(i int, wg *sync.WaitGroup, collection *Collection) error {

		// Store the document analysis
		if debug {
			fmt.Printf("\tLaunchAnalyseDocument - %v -  Storing the FileName:%v\n", i, doc.FileName)
		}

		err = doc.StoreDocumentAnalysis(collection.Conf)
		if err != nil {
			log.Printf("\tLaunchAnalyseDocument - %v - Error in storing FileName:%v - Error:%v\n", i, doc.FileName, err)
		}

	}

	return err
}

// contentAnalyserLauncher - Analyse content using ContentAnalyser Interface
// This is the effective wrapper fot calling the Analyser to process the document
// Uses channels to communicate the results of the analysis from the analyser to the launcher
func (doc *Document) contentAnalyserLauncher(debug bool) error {

	var err error

	analyser := doc.Analyser

	// Check if Analyser is disabled and skip the analysis is true
	if analyser.AnalyserDisabled() {
		fmt.Printf("\n\tAnalyseDocument - DEBUG - Skipping OpenAI analysis OPENAI_DISABLE is set for Document - FileName:%v\n", doc.FileName)
		return err
	}

	// Update - the number of text chunks
	doc.NumTextChunks = len(doc.TextChunks)

	tdmd := make(map[string][]ParagraphMetaData)
	doc.TypeDocMetaData = tdmd

	doc.AnalysisStats.ToProcess = doc.NumTextChunks
	doc.NumTextChunks = len(doc.TextChunks)

	if doc.NumTextChunks == 0 {
		doc.AnalysisStats.Warnings++
		return errors.New("AnalyseDocument - No text chunks found")
	}

	// Create a new Content Analyser instance for the document
	// This object is used to store the results of the analysis from the package
	conAnal := analyser.NewContentAnalysis(doc.EratoContentID, doc.TextChunks)

	// Run the Document Analyser - which then
	err = conAnal.AnalyseContent()
	if err != nil {
		// Handle the error
		doc.AnalysisStats.Errors++
	} else {
		doc.AnalysisStats.Processed++
	}

	// Loop through the results getting the results and errors
	analysisResultCount := conAnal.AnalysisResultCount()
	for i := 0; i < analysisResultCount; i++ {

		// Error handing from the Analysis
		// if conAnal.AnalysisResultError(i) != nil {
		if conAnal.AnalysisErrorCount() > 0 {

			// Increment the error count
			doc.AnalysisStats.Errors++
			doc.AnalysisErrors = append(doc.AnalysisErrors, conAnal.AnalysisResultError(i))

			if debug {
				fmt.Printf("\tAnalyseDocument - ERROR - %v\n", conAnal.AnalysisResultError(i))
			}

			// Error so don't add to the results
			continue

		} else {
			// Assume the analysis is a success
			doc.AnalysisStats.Success++
		}

		// Get the Analysis data from the content analyser
		ad := conAnal.AnalysisResultData(i)

		// Add AnalysisData for the Text Chunk to the document
		doc.DocMetaData = append(doc.DocMetaData, ad)

		// Add MetaData for the Paragram by type
		// doc.TypeDocMetaData[pmd.ParagraphType] = append(doc.TypeDocMetaData[pmd.ParagraphType], pmd)

	}

	// Print a new line to deal with the dots
	fmt.Printf("\n")

	// Post Analysis
	// keep text chunks if in debug mode ?
	if !debug {
		doc.TextChunks = nil
	}

	// Clean up the doc with processing references
	doc.DocumentData = nil
	doc.ContentRef = nil
	doc.Analyser = nil
	doc.Collector = nil

	// Set the document as curated
	doc.Curated = true

	if debug {
		fmt.Printf("\n\tAnalyseDocument - Completed Document:%v - Processed:%v - Success:%v - Errors:%v - Warnings:%v\n",
			doc.FileName,
			doc.AnalysisStats.Processed,
			doc.AnalysisStats.Success,
			doc.AnalysisStats.Errors,
			doc.AnalysisStats.Warnings)
	}

	return err

}

func (doc *Document) RxeportDocumentAnalysisStats() {
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

// StoreDocumentAnalysis - Save a document to the output directory
func (doc *Document) StoreDocumentAnalysis(c *Conf) error {
	// create a clean file name with a Unique ID to ensure that duplciate file names don't over-write each other
	// Using the path has of the full filename path
	if doc.Location == "" || doc.PathHash == "" || doc.FileName == "" {
		log.Fatal("StoreDocumentAnalysis")
	}

	fileBase := strings.ReplaceAll(filepath.Base(doc.FileName), " ", "_")
	fileBase = strings.ReplaceAll(fileBase, "/", "_")

	outFn := doc.Location + "-" + doc.PathHash + "-" + fileBase + ".json"
	outffn := filepath.Join(c.OutputDir, outFn)

	// open a file and wrire the contents

	// write the json to the file
	if c.Debug {
		fmt.Printf("\nStoreDocumentAnalysis - Content Catalog:%v - Document Name:%v FileName:%v - Writing to file:%v\n",
			doc.ContentSource,
			doc.Name,
			doc.FileName,
			outffn)
	}

	out := utils.PrettyStructDebug(doc)
	f, err := os.Create(outffn)
	if err != nil {
		log.Fatal(fmt.Errorf("StoreDocumentAnalysis - os.Create Error:%v", err))
		return err
	}

	_, err = f.WriteString(out)
	if err != nil {
		log.Fatal(fmt.Errorf("StoreDocumentAnalysis - os WriteString Error:%v", err))
		return err
	}

	return nil
}
