package erato

import (
	"Erato/erato/models"
	content "Erato/erato/preparers/content"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type ContentCatalog []Document

// MakeContentCatalog - Convert from a catalog of Content Source files to a Catalog of Erato Documents
// Filtering out folders and supported file types
func (collection *Collection) MakeEratoContentCatalog() (ContentCatalog, error) {

	var err error
	var cc []Document

	contentSource := collection.ContentSource
	c := collection.Conf

	// Setup the Content Source
	scc := contentSource.Collector

	// Loop through the content source and convert to a erato.Document
	// Iterate through the Content References
	for _, file := range scc.AllContentRefs() {

		// TODO -  Replace here with newDocument
		var doc Document

		// Add the analyser and collectors to the document for later processing
		// doc.Analyser = collection.ContentAnalyser
		// doc.Collector = collection.ContentSource.Collector

		// Add the Analyser to the Document

		// Map and update all the Metadata from the ContentRef to the Erato Document
		err := doc.UpdateEratoDocumentMetaData(file, collection)
		if err != nil {
			fmt.Printf("MakeEratoContentCatalog - Update Meta Data Error:%v", err)
			continue
		}

		// Update the Erato doc with the file extension and file supported types
		err = doc.UpdateType(collection)
		if err != nil {
			if c.Debug {
				fmt.Printf("MakeEratoContentCatalog - DEBUG - Warning - Document Not supported filename:%v - Error:%v\n", doc.FileName, err)
				doc.AnalysisStats.Warnings++
			}
			continue
		}

		// Filter out the file paths that are not required
		if doc.FilterPath(c) {
			// TODO Warning for Filtered out the fole path.
			if c.Debug {
				fmt.Printf("MakeEratoContentCatalog - DEBUG - Path Filtered out:%v\n", doc.FileName)
			}
			continue
		}

		// TODO Redundent - Merge with - doc.UpdateType()
		// On Include the supported list of extensions
		// if doc.IncludeExtensions(c) {
		// 	// TODO Warning for Filtered out the fole path.
		// 	if c.Debug {
		// 		fmt.Printf("MakeEratoContentCatalog - DEBUG - Path Filtered out:%v\n", doc.FileName)
		// 	}
		// 	// Not a supported extension
		// 	continue
		// }

		// Add to the ContentCatalog
		cc = append(cc, doc)
	}

	return ContentCatalog(cc), err

}

// filerFiles - Filter files based on the config
func (doc *Document) FilterPath(c *Conf) bool {

	filePath := doc.FileName
	// contRef, _ := doc.ContentRef.(ContentRef)

	// Check if any of the parent directories contain the exceptions
	// Iterate throught the exclusions to see if the file path contains any of the exclusions
	// TODO - Move to generic filter
	// for _, excludePath := range c.SharePoint.SPexcludedPath {
	for _, excludePath := range c.ExcludedPath {

		// No exclude path set - one element with a nil value
		if len(c.ExcludedPath) <= 1 && c.ExcludedPath[0] == "" {
			continue
		}

		// A empty for the path means
		if excludePath != "" && strings.Contains(filepath.Dir(filePath), excludePath) {
			return true
		}
	}

	// If the file extension/type is in the list then don't filter filter is false

	return false
}

// MakeEratoDocument - Map Values from a ContentRef to a erato.Document
func (doc *Document) UpdateEratoDocumentMetaData(ff interface{}, collection *Collection) error {

	file, _ := ff.(models.ContentRef)

	// Error is a placeholder for any processing later
	var err error

	// Store the Content Source ContentRef in the Doc for collector methods
	doc.ContentRef = ff

	// Store the Content Source Name in the Doc For traceability
	doc.ContentSource = collection.ContentSource.Name

	doc.Analyser = collection.ContentAnalyser
	doc.Collector = collection.ContentSource.Collector

	// These functions are implemented in the ContentRef interface
	// Mapping the implementation of the ContentSource to the Erato Document
	doc.EratoContentID = uuid.NewString()
	doc.SourceID = file.GetUniqueID()
	doc.Name = file.GetName()
	doc.FileName = file.GetFileName()
	doc.Location = file.GetLocation()
	doc.Type = file.GetTypeName()
	doc.Path = file.GetPath()
	doc.PathHash = file.GetPathHash()
	doc.ParentLocation = file.GetParentLocation()

	// TODO - add checks to ensure the key values are set
	return err

}

/*
func (doc *Document) IncludeExtensions(c *Conf) bool {

	// filePath := doc.FileName
	contRef, _ := doc.ContentRef.(models.ContentRef)

	// Check if the file extension is in the inclusions
	// TODO - Check the logic here depends on order of the inclusions

	ext := contRef.GetTypeName()
	// for _, inclusion := range c.SharePoint.SPincludedFileExtensions {
	for _, inclusion := range c.IncludedFileExtensions {
		if ext == inclusion {
			// filter = false
			return false
		} else {
			// filter = true
			return true
		}
	}

	return false
}
*/

// UpdateType - Determines what type and establishes to then be used by the preparer
func (doc *Document) UpdateType(collection *Collection) error {
	// return error if doc.FileName is empty
	if doc.FileName == "" {
		return errors.New("UpdateFileExtention - doc.FileName is empty")
	}

	contRef, _ := doc.ContentRef.(models.ContentRef)
	doc.FileExt = contRef.GetTypeName()

	// Get the configuration of the content preparer
	contentConfig := collection.ContentPreparer

	// Now implement the concrete type to drive content Preperation
	// Determine file type from filename extension
	switch doc.FileExt {

	case ".docx":
		doc.FileExt = ".docx"
		doc.ContentType = content.DOCX{Config: contentConfig}

	case ".html":
		doc.FileExt = ".html"
		doc.ContentType = content.HTML{Config: contentConfig}

	case ".pdf":
		// Convert from PDF to text
		// doc.FileExt = ".pdf"
		// doc.ContentType = content.PDF{Config: contentConfig}

		return fmt.Errorf("unsupported file type: %v", doc.FileExt)
	default:
		return fmt.Errorf("unsupported file type: %v", doc.FileExt)
	}

	return nil

}
