package main

import (
	"Erato/erato"
	"fmt"

	"flag"
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
		log.Printf("Erato - Error loading .env file:%v Error:%v\n", fn, err)
		os.Exit(1)
	}

	/*
		// Read the YML config file
		conf2, err := erato.NewConf2("erato.yml")
		if err != nil {
			log.Fatal(err)
		}

		e2, err := erato.NewErato2(conf2)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Config=%v\n", e2)

	*/

	// New Erato Object
	e, err := erato.NewErato(env)
	if err != nil {
		log.Fatal(err)
	}

	fooColl := erato.Collection{Name: "Collection 1",

		ContentSource: erato.ContentSource{Name: "website collector",
			Collector: e.EratoCollectors.Website,
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

		// Dump the filenames if debug set
		if e.Conf.Debug {
			collection.DumpCatalogFileNames()
		}

		// Add the Found stats
		collection.ContentCatalogsStats.Found = len(collection.ContentCatalog)

		err = collection.AnalyseContentCatalog()
		if err != nil {
			// TODO - Replace with logging
			log.Fatal(err)
		}

		// Store the content catalog
		err = collection.StoreContentCatalog()
		if err != nil {
			// TODO - Replace with logging
			log.Fatal(err)
		}

	}

}
