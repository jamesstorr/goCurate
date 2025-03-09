package erato

import (
	"Erato/erato/analysers/openai"
	sharepoint "Erato/erato/collectors/sharepoint"
	website "Erato/erato/collectors/website"
	content "Erato/erato/preparers/content"
	"Erato/erato/utils"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// STRUCTURES
// Config - Configuration for the Erato Application
type EratoConf struct {
	Conf       Conf2          `yaml:"Conf"`
	Collectors CollectorsConf `yaml:"Collectors"`
	Analysers  AnalysersConf  `yaml:"Analysers"`
	Preparer   PreparerConf   `yaml:"Preparer"`
}

type Conf2 struct {
	Debug           bool     `yaml:"Debug"`
	ExcludePaths    []string `yaml:"ExcludePaths"`
	AuditDir        string   `yaml:"AuditDir"`
	LogDir          string   `yaml:"LogDir"`
	AnalysisWorkers int      `yaml:"AnalysisWorkers"`
	DepthLimit      int      `yaml:"DepthLimit"`
}

type CollectorsConf struct {
	Sharepoint SharepointConf `yaml:"Sharepoint"`
	Website    WebsiteConf    `yaml:"Website"`
}

type SharepointConf struct {
	Name        string `yaml:"Name"`
	SecretsFile string `yaml:"SecretsFile"`
	SiteUrl     string `yaml:"SiteUrl"`
	DebugDepth  int    `yaml:"DebugDepth"`
	Debug       bool   `yaml:"Debug"`
}

type WebsiteConf struct {
	Name           string `yaml:"Name"`
	AllowedDomains string `yaml:"AllowedDomains"`
	Debug          bool   `yaml:"Debug"`
}

type AnalysersConf struct {
	OpenAI            OpenAIConf            `yaml:"OpenAI"`
	ComprehendMedical ComprehendMedicalConf `yaml:"ComprehendMedical"`
}

type OpenAIConf struct {
	Name        string `yaml:"Name"`
	BaseURL     string `yaml:"BaseURL"`
	SecretsFile string `yaml:"SecretsFile"`
	Model       string `yaml:"Model"`
	MaxTokens   int    `yaml:"MaxTokens"`
	Temp        int    `yaml:"Temp"`
	Workers     int    `yaml:"Workers"`
	PromptFile  string `yaml:"PromptFile"`
	Debug       bool   `yaml:"Debug"`
}

type ComprehendMedicalConf struct {
	Name        string `yaml:"Name"`
	ApiEndPoint string `yaml:"ApiEndPoint"`
	SecretsFile string `yaml:"SecretsFile"`
	Debug       bool   `yaml:"Debug"`
}

type PreparerConf struct {
	MaxParagraphWordCount int  `yaml:"MaxParagraphWordCount"`
	MinParagraphWordCount int  `yaml:"MinParagraphWordCount"`
	Debug                 bool `yaml:"Debug"`
}

func NewConf2(cf string) (*EratoConf, error) {
	var err error
	var conf EratoConf

	v := viper.New()
	v.SetConfigName(cf)
	v.SetConfigType("yml")

	// Set the path to look for the configurations file
	v.AddConfigPath("./config/")
	v.AddConfigPath("./secrets/")
	v.AddConfigPath("./")

	err = v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading config file %v,Error=%s", cf, err)
	}

	err = v.Unmarshal(conf)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshall configfile into struct, %v", err)
	}

	// Add value checker
	// err = CheckConfig("GlobalConf", gc)
	// if err != nil {
	// 	return err
	// }

	return &conf, err

}

type Conf struct {
	Env                    string
	ExcludedPath           []string
	IncludedFileExtensions []string
	Version                string
	LogDir                 string
	AuditDir               string
	OutputDir              string
	EratoAnalysisWorkers   int // ERATO_ANALYSIS_PARRALEL_REQUESTS
	Debug                  bool
	// To be depricated
	SharePoint      sharepoint.SharePointConfig
	Website         website.WebsiteConfig
	ContentPreparer content.Config
	XX_OAI          openai.Config
}

// NewConfig - Create a new config struct
func NewConfig() *Conf {

	// TODO - Bolt this back in
	// err := checkEnvVars(EnvVars)
	// if err != nil {
	// 	panic(err)
	// }

	// convert a string to an int
	max, err := strconv.Atoi(os.Getenv("PARAGRAPH_MAX_WORD_COUNT"))
	if err != nil {
		panic(err)
	}
	min, err := strconv.Atoi(os.Getenv("PARAGRAPH_MIN_WORD_COUNT"))
	if err != nil {
		panic(err)
	}

	levelLimit, err := strconv.Atoi(os.Getenv("LEVEL_LIMIT"))
	if err != nil {
		panic(err)
	}

	oaiParralelReq, err := strconv.Atoi(os.Getenv("OPENAI_WORKERS"))
	if err != nil {
		panic(err)
	}
	// Get the int value for the setting
	eratoAnalsysisWorkers, err := strconv.Atoi(os.Getenv("ERATO_ANALYSIS_WORKERS"))
	if err != nil {
		panic(err)
	}

	oaiMaxTokens, err := strconv.Atoi(os.Getenv("OPENAI_MAX_TOKENS"))
	if err != nil {
		panic(err)
	}

	oaiWorkerDelay, err := strconv.Atoi(os.Getenv("OPENAI_SLEEP"))
	if err != nil {
		panic(err)
	}

	// oaiTemp, err := strconv.Atoi(os.Getenv("OPENAI_TEMP"))
	oaiTemp, err := strconv.ParseFloat(os.Getenv("OPENAI_TEMP"), 32)
	if err != nil {
		panic(err)
	}

	debug := utils.StringToBool(os.Getenv("DEBUG"))

	oiac := openai.Config{
		OAIdisable:          utils.StringToBool(os.Getenv("OPENAI_DISABLE")),
		OAIapibase:          os.Getenv("OPENAI_BASE"),
		OAIapiKey:           os.Getenv("OPENAI_KEY"),
		OAImodel:            os.Getenv("OPENAI_MODEL"),
		OAImaxTokens:        oaiMaxTokens,
		OAItemperature:      float32(oaiTemp),
		OAIexampleFile:      os.Getenv("PROMPT_EXAMPLE_FILE"),
		OAIparralelRequests: oaiParralelReq,
		OpenAIworkerDelay:   oaiWorkerDelay,
		OIAprompt:           utils.Prompt(os.Getenv("PROMPT_EXAMPLE_FILE")),
		Debug:               debug,
	}

	spc := sharepoint.SharePointConfig{
		SPdepthLimit: levelLimit,
		SPsiteURL:    os.Getenv("SP_SITE_URL"),
		SPsiteName:   os.Getenv("SP_SITE_NAME"),
		SPAuthFile:   os.Getenv("SP_AUTH_FILE"),
		Debug:        debug,
	}

	web := website.WebsiteConfig{
		URL:            os.Getenv("WEBSITE_URL"),
		AllowedDomains: strings.Split(os.Getenv("WEBSITE_ALLOWED_DOMAINS"), ","),
		MaxDepth:       levelLimit,
		Debug:          debug,
	}

	cp := content.Config{
		ParagraphMaxWordCount: max,
		ParagraphMinWordCount: min,
		Debug:                 debug,
	}

	ExcludedPath := strings.Split(os.Getenv("EXCLUDED_PATHS"), ",")
	IncludedFileExtensions := strings.Split(os.Getenv("INCLUDED_FILE_EXTENSIONS"), ",")

	c := Conf{
		Env:                    os.Getenv("ENV"),
		ExcludedPath:           ExcludedPath,
		IncludedFileExtensions: IncludedFileExtensions,
		Version:                os.Getenv("VCS"),
		LogDir:                 os.Getenv("LOG_DIR"),
		OutputDir:              os.Getenv("OUTPUT_DIR"),
		XX_OAI:                 oiac,
		SharePoint:             spc,
		Website:                web,
		ContentPreparer:        cp,
		EratoAnalysisWorkers:   eratoAnalsysisWorkers,
		Debug:                  debug,
	}

	return &c
}

// TODO
// CheckConfig - Check the config values
func CheckConfig(name string, c interface{}) error {
	var err error

	return err

}
