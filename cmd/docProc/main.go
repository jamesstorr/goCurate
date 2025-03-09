package main

import (
	erato "Erato/erato"
	sharepoint "Erato/erato/collectors/sharepoint"
	"sync"

	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	DefaultEnv = "DONOTUSE"
)

func main() {

	var err error
	var wg sync.WaitGroup
	wg.Add(1)

	// Define a string flag for the filename with a default value and a usage description,
	//  default is /sites/bids/Shared Documents/Case Studies/DVSA Driving Test Case Study- Clean _RC1.docx
	pflag.String("filename", "/sites/bids/Shared Documents/Case Studies/DVSA Driving Test Case Study- Clean _RC1.docx", "Path to the file")

	// Define a string flag for the environment files extension,
	// default is prod
	pflag.String("env", "prod", "Environment Extension")

	// TODO flag and default for the output directory

	// Parse the flags
	pflag.Parse()

	// Bind the flags to the Viper configuration
	viper.BindPFlags(pflag.CommandLine)

	// Get the filename from the Viper configuration
	fileName := viper.GetString("filename")
	// fileName = "/sites/bids/Shared%20Documents/Case%20Studies/Good%20Case%20Study%20Material.docx"

	// Create a new Erato document
	var doc erato.Document
	doc.FileName = fileName

	env := viper.GetString("env")
	fn := ".env_" + env

	// Load the environment variables from the .env file
	if err := godotenv.Load(fn); err != nil {
		log.Printf("No .env file found %v\n", fn)
		os.Exit(1)
	}

	// Setup Config from environment variables or flags
	e, err := erato.NewErato(env)
	if err != nil {
		log.Fatal(err)
	}

	// Force the output folder to be the current folder
	e.Conf.OutputDir = "./"

	// Temp Hack to create the content collction and source
	contentSource := "BJSS_Bid_SharePoint"
	collection := erato.Collection{Name: contentSource,

		ContentSource: erato.ContentSource{Name: contentSource,
			Collector: e.EratoCollectors.Sharepoint,
		},
		ContentPreparer: e.EratoPreparer,
		ContentAnalyser: e.EratoAnalysers.OpenAI,
		Conf:            e.Conf,
	}

	// Validate supported filetypes
	// err = doc.UpdateType(&collection)
	// if err != nil {
	//TODO  Only log a warning and continue
	// 	fmt.Printf("Document Not supported filename:%v - Error:%v\n", doc.FileName, err)
	// 	log.Println(err)
	// 	os.Exit(1)
	// }

	// Type Assert the sharepoint collector
	foo := collection.ContentSource.Collector
	coll := foo.(*sharepoint.SharePointColector)

	spFile, err := coll.GetFileDetails(doc.FileName)
	if err != nil {
		log.Fatal(err)
	}

	// Update the Erato Document MetaData using the sharepoint File Details
	doc.UpdateEratoDocumentMetaData(&spFile, &collection)

	doc.UpdateType(&collection)

	collection.ContentCatalog, err = collection.MakeEratoContentCatalog()
	if err != nil {
		log.Fatal(err)

	}

	// Analyse the document
	// err = doc.AnalyseDocument(&collection)
	err = doc.AnalyseDocument(1, &wg, &collection)
	if err != nil {
		log.Fatal(err)
	}

	// Add the Document to the Erato ContentCatalog
	// TODO - Function e.ContentCatalogs.AddDocument(&doc)
	collection.ContentCatalog = append(collection.ContentCatalog, doc)

	err = doc.StoreDocumentAnalysis(e.Conf)
	if err != nil {
		log.Fatal(err)
	}

	// Summarise the Proccessing Statistics
	doc.ReportDocumentAnalysisStats()

}
