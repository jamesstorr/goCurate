package main

import (
	erato "Erato/erato"

	"path/filepath"

	"Erato/erato/utils"
	"flag"
	"fmt"
	"log"

	"os"

	"github.com/joho/godotenv"
)

const (
	DefaultEnv = "DONOTUSE"
)

var env string

func init() {
	env = os.Getenv("ENV")
}

func main() {
	// var err error

	// Setup Config from environment variables or flags
	if env == "" {
		flag.StringVar(&env, "env", DefaultEnv, "Environment name to select the right .env file e.g. -name prod looks for .env.prod ")
		flag.Parse()
	}

	// if !isRunningInDockerContainer() {
	// loads values from .env into the system
	fn := ".env_" + env
	if err := godotenv.Load(fn); err != nil {
		log.Printf("No .env file found %v\n", fn)
		os.Exit(1)
	}

	// New Erato Object
	e, err := erato.NewErato(env)
	if err != nil {
		log.Fatal(err)
	}

	// Temp Hack to create the content collction and source
	fooColl := erato.Collection{Name: "BJSS Bid Documents",

		ContentSource: erato.ContentSource{Name: "BJSS_Bid_SharePoint",
			Collector: e.EratoCollectors.Sharepoint,
		},
		ContentPreparer: e.EratoPreparer,
		ContentAnalyser: e.EratoAnalysers.OpenAI,
		Conf:            e.Conf,
	}
	// Append the content to the collections
	e.ContentCollections = append(e.ContentCollections, fooColl)

	// Loop though all the content sources
	// Create a cataglog of all the content
	// take the results from the Source specific Catalog function and add to the Erato Content Catalog
	for _, collection := range e.ContentCollections {

		// Catalogue the contenst of the Content source
		fmt.Printf("Cataloging ContentSource=%v\n", collection.Name)

		// setup the content collector
		contentCollector := collection.ContentCollector()

		// Catalog the contents of a content source using the Source interface
		// Store the results in Source Catalog Structure specific to the source
		err := contentCollector.CatalogContents()
		if err != nil {
			log.Fatal(err)
		}

		// Erato specific function that takes sharepoint content and converts to erato structure
		collection.ContentCatalog, err = collection.MakeEratoContentCatalog()
		if err != nil {
			// TODO - Replace with logging
			log.Fatal(err)
		}

		// Add the Found stats
		collection.ContentCatalogsStats.Found = len(collection.ContentCatalog)

		// Dump the filenames if
		if e.Conf.Debug {
			collection.DumpCatalogFileNames()
		}

		err = collection.AnalyseContentCatalog()
		if err != nil {
			// TODO - Replace with logging
			log.Fatal(err)
		}

	}

	// write Erato Content Catalogue to a file
	// outffn := filepath.Join(e.Conf.OutputDir, "EratoContentCatalogue"+utils.DateTimeString()+".json")
	outffn := filepath.Join(e.Conf.OutputDir, "EratoContentCatalogue.json")
	fmt.Printf("Erato - Writing to file:%v\n", outffn)
	out := utils.PrettyStructDebug(e.ContentCollections)

	f, err := os.Create(outffn)
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString(out)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	fmt.Println("*___________________________* Processing End *_______________________________________*")
	fmt.Println("*____________________________________________________________________________________*")

}
