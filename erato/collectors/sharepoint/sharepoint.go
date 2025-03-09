package sharepoint

import (
	"Erato/erato/utils"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/koltyakov/gosip"
	"github.com/koltyakov/gosip/api"
	strategy "github.com/koltyakov/gosip/auth/azurecert"
)

// SharePoint Collector Object
type SharePointColector struct {
	SPsite SharePointSite
	// SubSites          []*SharePointSite // Treat them as-if they were a Full SP site
	SPdepthLimit             int
	SPAuthFile               string
	SPexcludedPath           []string
	SPincludedFileExtensions []string
	DocumentLibraries        []DocumentLibrary
	AllLibraryFiles          []File
	SubSites                 []*SharePointSite // Treat them as-if they were a Full SP site
	Debug                    bool
}

type SharePointConfig struct {
	SPsiteName               string
	SPsiteURL                string
	SPdepthLimit             int
	SPAuthFile               string
	SPexcludedPath           []string
	SPincludedFileExtensions []string
	Debug                    bool
}

type SharePointSite struct {
	SiteName string
	SiteURL  string
	// isSubSite bool
	spAPI  *api.SP
	Config *SharePointConfig
}

type DocumentLibrary struct {
	ID              string
	Name            string
	ParentWebURL    string
	Path            string
	FolderHierarchy Folder
	LibraryFiles    LibraryFiles
}

type LibraryFiles map[string]File

type Folder struct {
	Level             int
	I                 int
	FolderName        string
	Folders           []Folder
	Files             []File
	Exists            bool
	IsWOPIEnabled     bool
	ItemCount         int
	Name              string
	ProgID            string
	ServerRelativeURL string
	TimeCreated       time.Time
	TimeLastModified  time.Time
	UniqueID          string
	WelcomePage       string
	DocumentLibrary   string
}

type ContentCatalog []File

// File - Sharepoint File definiton
type File struct {
	I                 int
	UniqueID          string
	Name              string
	FileTypeName      string
	FileType          interface{}
	Title             string
	ContentTag        string
	Exists            bool
	Length            int
	Level             int
	LinkingURI        string
	LinkingURL        string
	ServerRelativeURL string
	TimeCreated       time.Time `json:"TimeCreated"`
	TimeLastModified  time.Time `json:"TimeLastModified"`
	// TmpData           []byte
	DocumentLibrary string
	spAPI           *api.SP
}

// ----------------------------------------------------------
// Interact with SharePoint
// ----------------------------------------------------------
// Create a new SharePoint Site object
func NewCollector(cc interface{}) (*SharePointColector, error) {
	var err error
	// Asert the config to the SharePointConfig
	c, ok := cc.(*SharePointConfig)
	if !ok {
		return nil, fmt.Errorf("NewSharePointSite - Error asserting config type")
	}

	// TODO - Maybe have to pass an interface of the config ?

	// TODO Make this a YML config file for each of the sharepoint sites.

	// sps := SharePointSite{
	spc := SharePointColector{
		SPsite: SharePointSite{
			SiteName: c.SPsiteName,
			// TODO - get sitename from config
			SiteURL: c.SPsiteURL,
			// spAPI:   sp,
			Config: c,
		},
		SPdepthLimit:             c.SPdepthLimit,
		SPAuthFile:               c.SPAuthFile,
		SPexcludedPath:           c.SPexcludedPath,
		SPincludedFileExtensions: c.SPincludedFileExtensions,
		Debug:                    c.Debug,
	}

	// Add the goSip api object to the collector
	spc.SPsite.spAPI, err = setupAPI(&spc)
	if err != nil {
		return nil, fmt.Errorf("NewSharePointCollector - unable to setup SP connection: %v", err)
	}

	// sps.GetSubSitesMock("Data/subSiteList.txt")

	return &spc, err

}

// Setup API connections to SharePoint
func setupAPI(c *SharePointColector) (*api.SP, error) {

	var err error

	client := setupClient(c)

	// Debug mode is set int he funcion
	setupHookHandlers(c, client)
	sp := api.NewSP(client)

	return sp, err
}

func setupClient(c *SharePointColector) *gosip.SPClient {
	authCnfg := &strategy.AuthCnfg{}
	// SPconfigPath := "./secrets/private.json"

	if c.SPAuthFile == "" {
		log.Fatalf("Sharepoint setupClient - has no value")
	}

	if err := authCnfg.ReadConfig(c.SPAuthFile); err != nil {
		log.Fatalf("Sharepoint setupClient - unable to read config file: %v - %v", c.SPAuthFile, err)
	}

	return &gosip.SPClient{AuthCnfg: authCnfg}

}

// CatalogContents - Of a Sharepoint site
// Determine the document Libraries on the site.
// Catalog the contents of the side by:
//
//	Recurse through the folder hierarchy (as deep as the config setting allows)
//	filtering the folder paths out as per the config.
//
// Consolidate the sharepoint site object with all the data
func (spc *SharePointColector) CatalogContents() error {
	var err error
	var pdl *DocumentLibrary
	var dlFiles []File

	// sps := spc.SPsite
	// Retrieve the Site Details * Test Connection
	err = spc.SPsite.SiteConfig()
	if err != nil {
		return fmt.Errorf("CatalogContents - Error occured getting site configuration: %v", err)
	}

	// sps.ListSubSitesMock("Data/subSiteList.txt")
	// Populate the Document Libraries in the sharepoint site object
	err = spc.UpdateDocumentLibaries()
	if err != nil {
		return fmt.Errorf("CatalogContents - Error occured getting document libraries: %v", err)

	}

	// iterate through the document libraries and get the contents and update the LibraryFiles slice for a flat list
	for i := range spc.DocumentLibraries {

		// check a comma delimited list of libraries to include
		// TODO - move list to a constant
		if !strings.Contains("Documents,Site Assets,Translation Packages", spc.DocumentLibraries[i].Name) {
			continue
		}

		// Create a pointer to sps so the iterated Document library is updated
		pdl = &spc.DocumentLibraries[i]

		fmt.Printf("CatalogContents for Document Library-%v\n", pdl.Name)

		// Catalog the contents (files and folders) in the document library
		err := pdl.catalogDocumentLibraryContents(spc)
		if err != nil {
			return fmt.Errorf("CatalogContents - Error occured getting document library contents: %v", err)

		}

		// Consolidate the list of files to the sharepoint site
		for _, file := range pdl.LibraryFiles {
			dlFiles = append(dlFiles, file)
		}

		// Append slice of files to all the files
		spc.AllLibraryFiles = append(spc.AllLibraryFiles, dlFiles...)

		// Nil dlFiles for the next Loop
		dlFiles = nil

	}

	return err

}

// get Site configuration
func (sps *SharePointSite) SiteConfig() error {
	c := sps.Config
	spAPI := sps.spAPI

	// Get the Site Title
	// res, err := sp.Web().Select("Title").Get()
	res, err := spAPI.Web().Get()
	if err != nil {
		return fmt.Errorf("GetSiteName - unable to get title: %v", err)
	}

	if res.Data().Title == "" {
		return fmt.Errorf("GetSiteName - unable to get Site Name")
	}

	sps.SiteURL = res.Data().URL
	// sps.SiteConfiguration = res.Data().Configuration

	if c.Debug {
		fmt.Printf("SiteConfig -wor Site Name=%v\n", res.Data().Title)
	}

	return err

}

// SPdocumentLibaries - list all the document libraries in a SharePoint site
func (spc *SharePointColector) UpdateDocumentLibaries() error {
	sps := spc.SPsite
	c := sps.Config

	// Get the libaraies in the SharePoint site
	// This code retrieves only the lists of type "DocumentLibrary" by including a filter that checks the `BaseTemplate` property of each list.
	// The `101` value corresponds to the "Document Library" template in SharePoint.
	// libraries, err := sp.Web().Lists().Filter("BaseTemplate eq 101").Get()
	libraries, err := sps.spAPI.Web().Lists().Filter("BaseTemplate eq 101").Get()
	if err != nil {
		return fmt.Errorf("DocumentLibaries - unable to get lists: %v", err)
	}

	// Work through the libraries and add them to the SharePointSite struct
	for _, list := range libraries.Data() {
		var dl DocumentLibrary
		foo := list.Data()

		dl.ID = foo.ID
		dl.Name = foo.Title
		dl.ParentWebURL = foo.ParentWebURL
		dl.Path = getLibraryPath(foo.ParentWebURL, foo.DocumentTemplateURL, foo.Title)
		spc.DocumentLibraries = append(spc.DocumentLibraries, dl)

		if c.Debug {
			utils.PrintPrettyStructDebug(dl)
		}
	}

	return err

}

// getLibraryPath - VERY HACKY aAF - get the path to the library
func getLibraryPath(siteURL string, fURL string, title string) string {
	if fURL == "" {
		return title
	}
	// Remove the variable parts of the URL
	remainder := strings.TrimPrefix(fURL, siteURL)

	// Hacky AF
	split := strings.Split(remainder, "/")
	if len(split) < 2 {
		return title
	} else {
		return split[1]
	}
}

// // TODO add New  - list all the files and folders in a SharePoint Document library
func (dl *DocumentLibrary) catalogDocumentLibraryContents(spc *SharePointColector) error {

	var root Folder
	sps := spc.SPsite
	c := sps.Config
	libFiles := make(LibraryFiles)
	dl.LibraryFiles = libFiles

	err := getFilesAndFolders(c, sps.spAPI, dl, &root, 0)
	if err != nil {
		return fmt.Errorf("LibraryContents - Error occured get files and folders: %v", err)
	}

	dl.FolderHierarchy = root

	return err

}

// getFilesAndFolders - Allow the recursion to get the files and folders in a SharePoint library
func getFilesAndFolders(c *SharePointConfig, sp *api.SP, dl *DocumentLibrary, folder *Folder, level int) error {
	var err error
	var self api.FolderResp
	path := dl.Path

	// Get self folder details to determine the folder UniqueID
	// The initial level will not have got a UID yet so use the path
	if level == 0 {
		self, err = sp.Web().GetFolder(path).Get()

		// for all other levels use the UID to avoid path length issues
	} else {
		self, err = sp.Web().GetFolderByID(folder.UniqueID).Get()
	}
	if err != nil {
		return fmt.Errorf("getFilesFolders - Can't get self in path:%v - error:%v", path, err)
	}

	// Get and set the UID for the passed folder so UID can be used to get values.
	selfUID := self.Data().UniqueID
	if selfUID == "" {
		return fmt.Errorf("getFilesFolders - No Self UID returned for path:%v", path)
	}
	folder.UniqueID = selfUID

	// Internal helps to get the folders in the Library
	err = getFolders(c, sp, path, folder, dl, level)
	if err != nil {
		return fmt.Errorf("LibraryContents - collect Folder in %v - error:%v", path, err)
	}

	// Get the files in the folder
	err = getFiles(c, sp, folder, dl)
	if err != nil {
		return fmt.Errorf("LibraryContents - collect Files in %v - error:%v", path, err)
	}

	return err

}

func getFolders(c *SharePointConfig, sp *api.SP, path string, folder *Folder, dl *DocumentLibrary, level int) error {

	// Not Using GetFolderByPath() as this hits path length issues
	spFolders, err := sp.Web().GetFolderByID(folder.UniqueID).Folders().Get()
	if err != nil {
		return fmt.Errorf("getFolders - Error getting files:%v", err)
	}

	if len(spFolders.Data()) == 0 {
		if c.Debug {
			fmt.Printf("getFolders No folders found in %v\n", path)
		}
	}

	for i, spFolder := range spFolders.Data() {
		var internalFolder Folder
		if c.Debug {
			fmt.Printf("Folder i:%v name:%v date:%v\n", i, spFolder.Data().Name, spFolder.Data().TimeLastModified)
		}
		internalFolder.I = i
		internalFolder.FolderName = spFolder.Data().Name
		internalFolder.ServerRelativeURL = spFolder.Data().ServerRelativeURL
		internalFolder.ItemCount = spFolder.Data().ItemCount
		internalFolder.UniqueID = spFolder.Data().UniqueID
		internalFolder.ProgID = spFolder.Data().ProgID
		internalFolder.WelcomePage = spFolder.Data().WelcomePage
		internalFolder.Exists = spFolder.Data().Exists
		internalFolder.DocumentLibrary = dl.Name

		if c.Debug {
			fmt.Printf("getFolders-Level:%v-Recurse into folder:%v-path:%v\n", i, spFolder.Data().Name, spFolder.Data().ServerRelativeURL)
		}

		// Increment level to for the recursion to track the depth
		level++
		// limit the depth of the recursion
		if c.SPdepthLimit != 0 {
			if level > c.SPdepthLimit {
				continue
			}
		}

		// Recurse the files and folders to drill down into this loops folder
		err = getFilesAndFolders(c, sp, dl, &internalFolder, level)
		if err != nil {
			return fmt.Errorf("getFolders - Recurse Error occured get files and folders: %v", err)
		}

		// Append the folder to the parent folder
		folder.Folders = append(folder.Folders, internalFolder)
	}

	return err
}

// getFiles - list all the files in a SharePoint library Folder
func getFiles(c *SharePointConfig, sp *api.SP, folder *Folder, dl *DocumentLibrary) error {

	var files []File
	var err error

	// TODO - Add handling of 403 errors give a warning not a fatal error
	spFiles, err := sp.Web().GetFolderByID(folder.UniqueID).Files().Get()
	if err != nil {
		return fmt.Errorf("LibraryContents Error getting files error:%v", err)
	}

	// Debug mode report no files found
	if len(spFiles.Data()) == 0 {
		if c.Debug {
			fmt.Println("Files - No files found")
		}
	}

	// Iterate over the files listed returned.
	for i, spFile := range spFiles.Data() {
		// var file File
		// TODO

		file := mapFileValues(&spFile)

		file.I = i

		// Append file details to to the list of files
		files = append(files, file)
		// Add into the Library files map for the top level Document Library
		// so there is a simpler single map of all the files in the DL
		dl.LibraryFiles[file.UniqueID] = file
	}

	folder.Files = files

	return err

}

// GetFile - list all the files in a SharePoint library Folder
func (spc *SharePointColector) GetFileDetails(FullfilePath string) (File, error) {
	var file File
	var err error

	// Legacy Hangover Code
	sps := spc.SPsite
	c := sps.Config

	sp := sps.spAPI

	// Determine if the file with that path exists in the document library
	spFile, err := sp.Web().GetFileByPath(FullfilePath).Get()
	if err != nil {
		return file, fmt.Errorf("GetFile - GetFileByPath - Error:%v", err)
	}

	file = mapFileValues(&spFile)

	file.spAPI = sps.spAPI

	if c.Debug {
		fmt.Printf("Sharepoint File Details:\n%v\n", utils.PrettyStructDebug(file))
	}

	return file, err
}

// Helpe to map the file values from the sharepoint API to the File struct
func mapFileValues(spFile *api.FileResp) File {
	var file File

	file.Name = spFile.Data().Name
	file.FileTypeName = filepath.Ext(spFile.Data().Name)
	file.UniqueID = spFile.Data().UniqueID
	file.Title = spFile.Data().Title
	file.ContentTag = spFile.Data().ContentTag
	file.Exists = spFile.Data().Exists
	file.Length = spFile.Data().Length
	file.Level = spFile.Data().Level
	file.LinkingURI = spFile.Data().LinkingURI
	file.LinkingURL = spFile.Data().LinkingURL
	file.ServerRelativeURL = spFile.Data().ServerRelativeURL
	file.TimeCreated = spFile.Data().TimeCreated
	file.TimeLastModified = spFile.Data().TimeLastModified
	// file.DocumentLibrary = dl.Name

	return file
}

// DownloadContentData - provide the fileID to download the file for a file to avoid long file name
func (spc *SharePointColector) DownloadContentData(ff interface{}) (*[]byte, error) {

	// Hack to refactor
	sps := spc.SPsite

	// assert the file type
	file, ok := ff.(*File)
	if !ok {
		return nil, fmt.Errorf("DownloadContentData - Error asserting file type")
	}

	spAPI := sps.spAPI
	if file.UniqueID == "" {
		return nil, fmt.Errorf("\n\tDownloadLibraryFile - File ID is blank not found in library for file:%v", file.Name)
	}

	data, err := spAPI.Web().GetFileByID(file.UniqueID).Download()
	if err != nil {
		return nil, fmt.Errorf("DownloadLibraryFile - Error downloading file:%v", err)
	}

	if sps.Config.Debug {
		fmt.Printf("\n\tDownloadLibraryFile - Downloaded:%v - size:%v\n", filepath.Base(file.Name), len((data)))
	}

	return &data, err

}

/*
// DownloadContentData - provide the fileID to download the file for a file to avoid long file name
func (file *File) DownloadContentData2() (*[]byte, error) {

	// Get the content

	spAPI := file.spAPI

	// spAPI := sps.spAPI
	if file.UniqueID == "" {
		return nil, fmt.Errorf("\n\tDownloadLibraryFile - File ID is blank not found in library for file:%v", file.Name)
	}

	data, err := spAPI.Web().GetFileByID(file.UniqueID).Download()
	if err != nil {
		return nil, fmt.Errorf("DownloadLibraryFile - Error downloading file:%v", err)
	}

	// if sps.Config.Debug {
	// 	fmt.Printf("\n\tDownloadLibraryFile - Downloaded:%v - size:%v\n", filepath.Base(file.Name), len((data)))
	// }

	return &data, err

}
*/

// filerFiles - Filter files based on the config
func FilterFilePath(c *SharePointConfig, filePath string) bool {

	// Check if any of the parent directories contain the exceptions
	for _, excludePath := range c.SPexcludedPath {
		if strings.Contains(filepath.Dir(filePath), excludePath) {
			return true
		}
	}

	// If the file extension/type is in the list then don't filter filter is false
	// Check if the file extension is in the inclusions

	// Check the logic here depends on order of the inclusions
	ext := filepath.Ext(filePath)
	for _, inclusion := range c.SPincludedFileExtensions {
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

func ListsubSites(c *SharePointConfig) {

	/*
		spAPI, err := setupAPI(c)
		if err != nil {
			fmt.Println("ListSPsubSites - Error setting up API:", err)
			return
		}

		// Get the subsites in the SharePoint site
		subsites, err := spAPI.Webs().Select("Title").Get()
		if err != nil {
			fmt.Println("Error getting subsites:", err)
			return
		}

		// Print the titles of all the subsites
		for _, subsite := range subsites.Data() {
			fmt.Println(subsite.Data().Title)
		}

	*/

}

func SubSiteName(c *SharePointConfig) {

	/*
		var err error

		spAPI, err := setupAPI(c)
		if err != nil {
			fmt.Println("ListSPsubSites - Error setting up API:", err)
			return
		}

		// res, err := spAPI.Web().Select("Title").Get()
		res, err := spAPI.Webs().Select("Title").Get()
		if err != nil {
			log.Fatalf("GetFilesSharepoint - unable to get title: %v", err)
		}

		fmt.Printf("Site Name=%v", res.Data())
	*/

}

// Setup hook handlers for the api
func setupHookHandlers(c *SharePointColector, client *gosip.SPClient) {
	// Define requests hook handlers
	// TODO Add logging output to a file

	if c.Debug {

		client.Hooks = &gosip.HookHandlers{
			OnError: func(e *gosip.HookEvent) {
				fmt.Println("\n======= On Error ========")
				fmt.Printf(" URL: %s\n", e.Request.URL)
				fmt.Printf(" StatusCode: %d\n", e.StatusCode)
				fmt.Printf(" Error: %s\n", e.Error)
				fmt.Printf("  took %f seconds\n",
					time.Since(e.StartedAt).Seconds())
				fmt.Printf("=========================\n\n")
			},
			OnRetry: func(e *gosip.HookEvent) {
				fmt.Println("\n======= On Retry ========")
				fmt.Printf(" URL: %s\n", e.Request.URL)
				fmt.Printf(" StatusCode: %d\n", e.StatusCode)
				fmt.Printf(" Error: %s\n", e.Error)
				fmt.Printf("  took %f seconds\n",
					time.Since(e.StartedAt).Seconds())
				fmt.Printf("=========================\n\n")
			},
			OnRequest: func(e *gosip.HookEvent) {
				if e.Error == nil {
					fmt.Println("\n====== On Request =======")
					fmt.Printf(" URL: %s\n", e.Request.URL)
					fmt.Printf("  auth injection took %f seconds\n",
						time.Since(e.StartedAt).Seconds())
					fmt.Printf("=========================\n\n")
				}
			},
			OnResponse: func(e *gosip.HookEvent) {
				if e.Error == nil {
					fmt.Println("\n====== On Response =======")
					fmt.Printf(" URL: %s\n", e.Request.URL)
					fmt.Printf(" StatusCode: %d\n", e.StatusCode)
					fmt.Printf("  took %f seconds\n",
						time.Since(e.StartedAt).Seconds())
					fmt.Printf("==========================\n\n")
				}
			},
		}

		//
	} else {

		client.Hooks = &gosip.HookHandlers{
			OnError: func(e *gosip.HookEvent) {
				fmt.Println("\n======= On Error ========")
				fmt.Printf(" URL: %s\n", e.Request.URL)
				fmt.Printf(" StatusCode: %d\n", e.StatusCode)
				fmt.Printf(" Error: %s\n", e.Error)
				fmt.Printf("  took %f seconds\n",
					time.Since(e.StartedAt).Seconds())
				fmt.Printf("=========================\n\n")
			},
			OnRetry: func(e *gosip.HookEvent) {
				fmt.Println("\n======= On Retry ========")
				fmt.Printf(" URL: %s\n", e.Request.URL)
				fmt.Printf(" StatusCode: %d\n", e.StatusCode)
				fmt.Printf(" Error: %s\n", e.Error)
				fmt.Printf("  took %f seconds\n",
					time.Since(e.StartedAt).Seconds())
				fmt.Printf("=========================\n\n")
			},
		}

	}
}

// Functions to implement the ContentRef interface
// AllContentRefs - return all the content references
func (spc *SharePointColector) AllContentRefs() []interface{} {
	// var retVal []*File
	var retVal []interface{}

	for i := range spc.AllLibraryFiles {

		// this specifc syntax is required to get the interface{} type into the slice
		ff := interface{}(&spc.AllLibraryFiles[i])
		retVal = append(retVal, ff)

	}

	return retVal
}

// Return the file type
func (file *File) ContentType() string {
	fp := file.GetPath()
	return filepath.Ext(fp)
}

func (file *File) GetUniqueID() string {
	return file.UniqueID
}

func (file *File) GetName() string {
	return file.Name
}

func (file *File) GetFileName() string {
	return file.ServerRelativeURL
}

func (file *File) GetLocation() string {
	return file.DocumentLibrary
}

func (file *File) GetTypeName() string {
	return file.FileTypeName
}

func (file *File) GetPath() string {
	return filepath.Dir(file.ServerRelativeURL)

}

func (file *File) GetPathHash() string {
	return utils.HashString(filepath.Dir(file.ServerRelativeURL))

}

// Legacy SubSite Code
/*

type mockSubSite struct {
	Name string
	URL  string
}

// ListSubSitesMock - write a function which opens a text file (parameter to the function) and reads the contents
// the contents are a list of SharePoint sub sites URLS
// each URL needs the string /Forms/AllItems.aspx removing from the end
// The is needs to be added as a SubSite struct as a slice to the SharePointSite struct field SubSites
func (sps *SharePointSite) GetSubSitesMock(filePath string) error {

	var mss []mockSubSite

	//
	if filePath == "" {
		return fmt.Errorf("GetSubSitesMock - No file path provided")
	}

	// Open the CSV file defined by filePath and read line by file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("GetSubSitesMockError opening file: %v", err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read the CSV records
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading record: %v\n", err)
			continue
		}

		// Skip blank URL Values
		if record[17] == "" {
			continue
		}

		mss = append(mss, mockSubSite{Name: record[4], URL: record[17]})
	}

	// Remove the /Forms/AllItems.aspx suffix from each URL
	var subSites []*SharePointSite
	for _, ss := range mss {

		subSite := SharePointSite{
			SiteURL:  strings.TrimSuffix(ss.URL, "/Forms/AllItems.aspx"),
			SiteName: ss.Name,
			spAPI:    sps.spAPI,
			// isSubSite: true
		}
		// Name: res.Data().Title,
		// Name: res.Name,
		// ID:   res.ID
		// subSites = append(subSites, &subSite)

	}

	// Add the data to the sharepoint site
	sps.SubSites = subSites

	return err
}

// ListSubSites - list all the sub sites in a SharePoint site
func (sps *SharePointSite) GetSubSites() error {
	spAPI := sps.spAPI

	// if err != nil {
	// 	return fmt.Errorf("GetSiteName - unable to setup SP connection: %v", err)
	// }

	// Get the Site Title
	// res, err := sp.Web().Select("Title").Get()
	res, err := spAPI.Web().Get()
	if err != nil {
		return fmt.Errorf("ListSubSites - GetSiteName - unable to get title: %v", err)
	}

	res.Data().Title = sps.SiteName

	webs, err := spAPI.Web().Webs().Get()
	if err != nil {
		return fmt.Errorf("ListSubSites - unable to get subsites: %v", err)
	}

	foo := webs.Data()

	utils.PrintPrettyStructDebug(foo)

	return err

}

*/
