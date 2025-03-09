package website

import (
	"Erato/erato/utils"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"path"
	"path/filepath"

	"github.com/gocolly/colly"
	"github.com/google/uuid"
)

// Using the Colly package to scrape a website
// download the content
// Website - Interface for the Website
type WebsiteCollector struct {
	Config         *WebsiteConfig
	SiteURL        string
	MaxDepth       int
	Colly          *colly.Collector
	AllowedDomains []string
	AllSitePages   []Page
	Debug          bool
}

type WebsiteConfig struct {
	URL            string
	AllowedDomains []string
	MaxDepth       int
	Debug          bool
}

type Page struct {
	I         int
	UniqueID  string
	Site      string
	URL       string
	ParentURL string
	Name      string
	TypeName  string
	Type      interface{}
	BodyData  []byte
}

func NewCollector(cc interface{}) (*WebsiteCollector, error) {
	var err error
	// Asert the config to the WebsiteConfig
	c, ok := cc.(*WebsiteConfig)
	if !ok {
		return nil, fmt.Errorf("NewWebsiteCollector - Error asserting config type")
	}

	// TODO - A config slice to create some optionality on Collector

	cly := colly.NewCollector(
		colly.AllowedDomains(c.AllowedDomains...),
		colly.MaxDepth(c.MaxDepth),
		colly.UserAgent(requestAgent()),
		// colly.Debugger(&debug.LogDebugger{}),
		// TODO attach a debugger to the collector if debug mode is set
	)

	// Limit the number of threads started by colly to two
	// when visiting links which domains' matches "*httpbin.*" glob
	// cly.Limit(&colly.LimitRule{
	// DomainGlob:  "*httpbin.*",
	// Parallelism: 2,
	// 	RandomDelay: 5 * time.Second,
	// })

	w := WebsiteCollector{
		Config:         c,
		SiteURL:        c.URL,
		Colly:          cly,
		AllowedDomains: c.AllowedDomains,
	}

	return &w, err
}

// requestAgent - Return a random user agent from the list
func requestAgent() string {

	x := []string{
		`Mozilla/5.0 (Linux; Android 13; SM-S908B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Mobile Safari/537.36`,
		`Mozilla/5.0 (iPhone14,3; U; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/602.1.50 (KHTML, like Gecko) Version/10.0 Mobile/19A346 Safari/602.1`,
		`Mozilla/5.0 (Linux; Android 13; Pixel 6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Mobile Safari/537.36`,
		`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36`,
		`Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0`,
		`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.2420.81`,
		`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 OPR/109.0.0.0`,
		`Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36`,
		`Mozilla/5.0 (Macintosh; Intel Mac OS X 14.4; rv:124.0) Gecko/20100101 Firefox/124.0`,
		`Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15`,
		`Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 OPR/109.0.0.0`,
		`Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36`,
		`Mozilla/5.0 (X11; Linux i686; rv:124.0) Gecko/20100101 Firefox/124.0`,
		`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"`,
	}

	// create a raondom number between 0 and the length of the slice of x-1
	return x[rand.Intn(len(x)-1)]

}

// CatalogContents - Catalog the contents of the website
func (w *WebsiteCollector) CatalogContents() error {
	var err error

	debug := w.Config.Debug

	// Setup the colly actions
	w.setupCollyBehaviour()
	if debug {
		fmt.Println("CatalogContents - SiteURL=", w.SiteURL)

	}
	// Start scraping on the site
	err = w.Colly.Visit(w.SiteURL)

	return err

}

// setupCatalogActions - Setup colly actions as it traverses the website
func (w *WebsiteCollector) setupCollyBehaviour() {

	debug := w.Config.Debug

	// Log the site vist
	w.Colly.OnRequest(func(r *colly.Request) {
		if debug {
			log.Println("Visiting", r.URL)
		}
		// Save the requested URL into context
		// r.Ctx.Put("Regurl", r.URL.String())

	})

	// Setup OnError behaviour for a page visit
	w.Colly.OnError(func(r *colly.Response, err error) {
		// Need to register connection
		log.Println("CatalogContents - ERROR:", r.StatusCode, err)
	})

	// The recursive link visiting function
	w.Colly.OnHTML("a[href]", func(e *colly.HTMLElement) {

		r := e.Response

		// Save the previous page URL into context
		parent := r.Request.URL.String()
		link := e.Attr("href")
		key := "ParentPage-" + e.Request.AbsoluteURL(link)
		r.Ctx.Put(key, parent)

		fmt.Printf("DEBUG - OnHTML - ParentPage:%v - Link:%v\n", parent, link)

		// if debug {
		// 	fmt.Printf("Link found:%s\n", link)
		// }
		// Visit link found on page Only those links are visited which are in AllowedDomains
		// sleep for a random amount of time betweek 100 and 500 milliseconds

		// time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond)
		e.Request.Visit(e.Request.AbsoluteURL(link))

	})

	// OnScraped event adds to the collectors list of objects
	w.Colly.OnScraped(func(r *colly.Response) {
		if debug {
			log.Println("Visited", r.Request.URL.String())
		}

		key := "ParentPage-" + r.Request.URL.String()
		prev := r.Ctx.Get(key)

		if prev == "" {
			fmt.Println("DEBUG - OnScraped - ParentPage is empty")
		}

		fmt.Printf("DEBUG - OnScraped - PreviousPage=%v\n", prev)

		// Create the Content of type Page
		page := Page{
			URL:       r.Request.URL.String(),
			ParentURL: prev,
			UniqueID:  uuid.NewString(),
			// TODO - less hacky and get actual content type
			TypeName: ".html",
			BodyData: r.Body,
		}

		page.addPageToCollection(w)

	})
}

// addPageToCollection - Add the page to the collection
func (p Page) addPageToCollection(w *WebsiteCollector) {

	// add to the list of successfuly collected pages
	w.AllSitePages = append(w.AllSitePages, p)

}

func (w *WebsiteCollector) DumpCatalogFileNames() {
	for _, p := range w.AllSitePages {
		fmt.Printf("DumpCatalogFileNames - Page=%v\n", p.URL)
	}

}

// AllContentRefs - Return all the content references
func (w *WebsiteCollector) AllContentRefs() []interface{} {
	var retVal []interface{}

	for i := range w.AllSitePages {

		// this specifc syntax is required to get the interface{} type into the slice
		ff := interface{}(&w.AllSitePages[i])
		retVal = append(retVal, ff)

	}

	return retVal
}

// Return the Content type
func (p *Page) ContentType() string {
	// TODO - HACKY HACK
	return ".html"
}

// DownloadContentData - Download the content data
func (w *WebsiteCollector) DownloadContentData(pr interface{}) (*[]byte, error) {
	p := pr.(*Page)
	// TODO - add check assertion

	// Return the body data which has already been captured by colly
	data := &p.BodyData
	return data, nil
}

// ContentRef - Interface for the Data that is sourced
// type ContentRef interface {
func (p *Page) GetUniqueID() string {
	return p.UniqueID

}
func (p *Page) GetName() string {
	// TODO - this maynot be index.html
	u, _ := url.Parse(p.URL)
	return path.Base(u.Path)

}
func (p *Page) GetFileName() string {
	// TODO - this maynot be index.html

	// log.Fatal("GetFileName - Not implemented")
	log.Printf("GetFileName - FIX This so there is no / for /index.htm - Not implemented")
	// FIX This so there is no / for /index.htm

	u, _ := url.Parse(p.URL)
	return path.Base(u.Path)
}

func (p *Page) GetLocation() string {
	u, _ := url.Parse(p.URL)
	return u.Hostname()
}
func (p *Page) GetTypeName() string {
	return p.TypeName
}

func (p *Page) GetPath() string {
	// return the path to the file from the URL
	// u, _ := url.Parse(p.URL)
	return p.URL

}

func (p *Page) GetPathHash() string {
	return utils.HashString(filepath.Dir(p.URL))
}

func (p *Page) GetParentLocation() string {
	return p.ParentURL
}
