package openai

import (
	"Erato/erato/models"
	"Erato/erato/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"

	openai "github.com/sashabaranov/go-openai"
)

type Config struct {
	OAIdisable          bool
	OAIapibase          string
	OAIapiKey           string
	OAImodel            string
	OAImaxTokens        int
	OAItemperature      float32
	OAIexampleFile      string
	OIAprompt           string
	OAIparralelRequests int
	OpenAIworkerDelay   int
	Debug               bool
}

// Change the name
type OpenAI struct {
	OAIdisable          bool
	OAIapibase          string
	OAIapiKey           string
	OAImodel            string
	OAImaxTokens        int
	OAItemperature      float32
	OAIexampleFile      string
	OIAprompt           string
	OAIparralelRequests int
	OpenAIworkerDelay   int
	Results             []AnalysisData
	Debug               bool
}

type ContentAnalysisData struct {
	OpenAI          *OpenAI
	DocID           string
	Content         []string
	AnalysisResults []Analysis
	AnalysisStats   ContentAnalysisStats
	AnalysisErrors  []error
}

// Depricated
// Extracted Entities from the Content - tied to the prompt
type AnalysisData_OLD struct {
	ParagraphNum      int
	ParagraphType     string   `json:"Paragraph Type"`
	ParagraphText     string   `json:"-"`
	ParagraphSummary  string   `json:"Paragraph Summary"`
	ParagraphQuestion string   `json:"Paragraph Question"`
	ClientName        []string `json:"Client Names"`
	Projects          []string `json:"Projects"`
	Technologies      []string `json:"Technologies"`
	Systems           []string `json:"Systems"`
	Methods           []string `json:"Methods"`
	PeopleNames       []string `json:"People Names"`
	OrganisationNames []string `json:"Organisation Names"`
	BusinessUnits     []string `json:"Business"`
	AnalysisError     error
	ResponseInfo      openai.ChatCompletionResponse
	// To be implemented
	// QuantativeInformation []string `json:"Quantative Information"`
	// ResultOutcome         string   `json:"Result or Outcome"`
}

type Analysis struct {
	AnalysisData
	AnalysisMetaData
}

type AnalysisData map[string]interface{}

// AnalysisMetaDaya - Meta Data for the Analysis
// Added as a key _EratoMetaData to the AnalysisData
type AnalysisMetaData struct {
	ParagraphNum  int
	AnalysisError error
	ResponseInfo  openai.ChatCompletionResponse
	// TODO: date and time, and other meta data
}

// TextChunkAnalysisResult - Stats for the Document Analysis
type TextChunkAnalysis struct {
	Order int
	// Ad    AnalysisData
	Analysis
	Err error
}

type ContentAnalysisStats struct {
	ToProcess int
	Processed int
	Success   int
	Errors    int
	Warnings  int
}

// NewOpenAIConfig - Create a new OpenAIConfig to be used by an instance of Content Analysis
func NewOpenAI(c *Config) (*OpenAI, error) {
	oai := OpenAI{
		OAIdisable:          c.OAIdisable,
		OAIapibase:          c.OAIapibase,
		OAIapiKey:           c.OAIapiKey,
		OAImodel:            c.OAImodel,
		OAImaxTokens:        c.OAImaxTokens,
		OAItemperature:      c.OAItemperature,
		OAIexampleFile:      c.OAIexampleFile,
		OIAprompt:           c.OIAprompt,
		OpenAIworkerDelay:   c.OpenAIworkerDelay,
		OAIparralelRequests: c.OAIparralelRequests,
		Debug:               c.Debug,
	}
	return &oai, nil
}

// Create a new instance of a Content Analysis Object
// This uses the OpenAI Object for the Analysis work
func (oai *OpenAI) NewContentAnalysis(EratoID string, content []string) models.ContentAnalysis {
	// func (oai *OpenAI) NewContentAnalysis(EratoID string, content []string) interface{} {

	cad := ContentAnalysisData{
		DocID:   EratoID,
		OpenAI:  oai,
		Content: content,
	}
	return &cad
}

// func (oai *OpenAI) AnalyseContent(textChunks []string) ([]interface{}, error) {
// AnalyseTextChunks - Analyse the text chunks and add the results to the Content Analysis Object
func (ca *ContentAnalysisData) AnalyseContent() error {

	var wg sync.WaitGroup
	var err error
	var a Analysis

	// TODO - Move to parameter
	// var OpenAIworkerDelay = 5000
	// OpenAIworkerDelay := ca.OpenAI.OpenAIworkerDelay
	// var OpenAIworkerDelay = 500

	debug := ca.OpenAI.Debug
	analyserWorkerCount := ca.OpenAI.OAIparralelRequests
	TextChunks := ca.Content
	analyser := ca.OpenAI

	// var chunkmetaData interface{}
	NumTextChunks := len(TextChunks)
	// Create a channel to receive the results
	textAnalysisResultChan := make(chan TextChunkAnalysis, NumTextChunks)

	// Set the worker numbers
	wrkNum := 0

	// Parrallelise the analysis of the text chunks
	for i, textChunk := range TextChunks {
		// Increment value of i so the first paragraph is 1 for traceability
		i++

		// Check to see if the number of workers has been reached
		if wrkNum == analyserWorkerCount {
			// Wait for the workers to finish
			wg.Wait()
			wrkNum = 0
		}

		wg.Add(1)

		ca.AnalysisStats.Processed++
		wrkNum++

		// Screen Feedback
		if debug {
			log.Printf("openai.AnalyseContent - DEBUG - DocID:%v Launching Paragraph %v - worker %v\n", ca.DocID, i, wrkNum)
			log.Printf("openai.AnalyseContent - DEBUG - DocID:%v Sleeping:%v Paragraph %v - worker %v\n", ca.DocID, ca.OpenAI.OpenAIworkerDelay, i, wrkNum)
		} else {
			fmt.Printf("%v", i)
		}
		// Fork the text processing, but implement the wrapper to handle go function timing issues
		// Add a sleep timer for starting the next worker of 0.5 of a second
		// to help the openAI API to handle incoming connections
		time.Sleep(time.Duration(ca.OpenAI.OpenAIworkerDelay) * time.Millisecond)

		// Now process the text chunk
		go func(textChunk string, i int) {
			analyseTextChunk(analyser, &textChunk, i, &wg, textAnalysisResultChan)
		}(textChunk, i)

		if debug {
			log.Printf("openai.AnalyseContent - DEBUG - DocID:%v Sleeping:%v Paragraph %v - worker %v\n", ca.DocID, ca.OpenAI.OpenAIworkerDelay, i, wrkNum)
		}
		time.Sleep(time.Duration(ca.OpenAI.OpenAIworkerDelay) * time.Millisecond)

	}

	// Add MetaData for the Paragram by type
	// doc.TypeDocMetaData[pmd.ParagraphType] = append(doc.TypeDocMetaData[pmd.ParagraphType], pmd)

	// Read all results from the result channel
	for range TextChunks {
		// read the fist item from the channel
		result := <-textAnalysisResultChan

		// Check for errors
		if result.Err != nil {
			// Increment the error count
			ca.AnalysisStats.Errors++

			// Add the error to the analysis data
			// result.Ad.AnalysisError = result.Err

			// Add a nill result so that the error handling processes it correctly
			ca.AnalysisResults = append(ca.AnalysisResults, Analysis{})

			// Add the error to the error list
			ca.AnalysisErrors = append(ca.AnalysisErrors, result.Err)

			if debug {
				log.Printf("\topenai.AnalyseContent - ERROR - %v\n", result.Err)
			} else {
				fmt.Printf("e")
			}
			// Error so don't add to the results
			continue

		} else {
			// Add to the success tally
			ca.AnalysisStats.Success++
		}

		// Add the Analysis Data to the Content Analysis Object
		a.AnalysisData = result.AnalysisData
		// TODO - Check that analysis is being added
		a.AnalysisMetaData.ParagraphNum = result.Order
		a.AnalysisMetaData.ResponseInfo = result.ResponseInfo

		// a.AnalysisMetaData.AnalysisError = result.Err

		// Add the Analysis Data to the Content Analysis Object
		ca.AnalysisResults = append(ca.AnalysisResults, a)
	}

	// Add MetaData for the Paragram by type
	// doc.TypeDocMetaData[result.Pmd.ParagraphType] = append(doc.TypeDocMetaData[result.Pmd.ParagraphType], result.Pmd)

	return err
}

// Change to conectSource orientated
func analyseTextChunk(analyser *OpenAI, textChunk *string, i int, wg *sync.WaitGroup, resultChan chan<- TextChunkAnalysis) {
	defer wg.Done()
	debug := analyser.AnalyserDebug()
	wordCount := len(strings.Fields(*textChunk))

	if debug {
		fmt.Printf("\t\topenai.analyseTextChunk - DEBUG - Starting analyseTextChunk Paragraph:%v Word Count:%v\n", i, wordCount)
	}

	// Extact the entities from the text chunk into a string
	ee, err := analyser.ExtractEntities(i, textChunk)
	if err != nil {
		// write the error back to the channel
		fmt.Printf("\t\topenai.analyseTextChunk - DEBUG - Ending with ERROR Paragraph:%v Error:%v\n", i, err)
		resultChan <- TextChunkAnalysis{Err: fmt.Errorf("openai.analyseTextChunk - Paragraph %v - Error %v", i, err)}
		return
	}

	// Marshall the Extracted Entities into the Analysis struct
	a, err := MarshallAnalysisData(ee)
	// Fake Error to test error handling
	// err = fmt.Errorf("FAKE Error - MarshallAnalysisData - %v", err)

	if err != nil {
		if debug {
			fmt.Printf("openai.analyseTextChunk - ERROR - Paragraph %v - Error=%v\n", i, err)
			fmt.Printf("openai.analyseTextChunk - ERROR - Paragraph %v - Extracted Entities=%v\n", i, ee)
			fmt.Printf("openai.analyseTextChunk - ERROR - Paragraph %v - debug values of extract=%v\n", i, utils.PrettyStructDebug(ee))
			fmt.Printf("\t\topenai.analyseTextChunk - DEBUG - Ending with ERROR Paragraph %v\n", i)
		}

		// Send the error result to the channel
		resultChan <- TextChunkAnalysis{Err: fmt.Errorf("openai.analyseTextChunk - Error - Paragraph %v - unable to Marshal into AnalysisData %v", i, err)}
		return
	}

	if debug {
		log.Printf("openai.analyseTextChunk - Paragraph %v - DEBUG START____________________________________________________________________________________\n\n", i)
		// log.Printf("analyseTextChunk - Paragraph %v - Text=%v\n", i, (*textChunk))
		log.Printf("openai.analyseTextChunk - Paragraph %v - Extracted Entities=%v\n", i, ee)
		log.Printf("openai.analyseTextChunk - Paragraph %v - METADATA=%v\n", i, utils.PrettyStructDebug(a))
		log.Printf("openai.analyseTextChunk - Paragraph %v - DEBUG END____________________________________________________________________________________\n\n", i)

		fmt.Printf("\t\topenai.analyseTextChunk - DEBUG - Ending Wrting to channel analyseTextChunk Paragraph %v\n", i)
	}

	// Send the result to the channel
	resultChan <- TextChunkAnalysis{Order: i, Analysis: a, Err: nil}
}

// Marshall Extract String - take the json text created openAI.ExtractEntities
// and marshall into a ParagraphMetaData struct to confirm with the data model
// MarshallAnalysisData - Marshall the Extracted Entities and metadata into the Analysis struct
func MarshallAnalysisData(eer ExtractEntitiesResponse) (Analysis, error) {
	var a Analysis
	ad := make(AnalysisData)

	// convert the extractString into a ParagraphMetaData struct
	err := json.Unmarshal([]byte(eer.extractEntitiesResponse), &ad)
	if err != nil {
		log.Printf("MarshallAnalysisData - Error - Can't Marshall Error-%v\n", err)
		log.Printf("MarshallAnalysisData - Debug OAI Chat response:\n%v\n",
			utils.PrettyStructDebug((&eer.Info)))

		return a, fmt.Errorf("MarshallAnalysisData Can't Marshall Error-%v", err)

	}

	// Add the Analysis Data into the Analysis Object
	a.AnalysisData = ad

	// Add in the Metadata into the analysis object
	a.AnalysisMetaData.ResponseInfo = storeChatResponseInfo(&eer.Info)

	return a, err
}

type ExtractEntitiesResponse struct {
	extractEntitiesResponse string
	// tokens
	Info openai.ChatCompletionResponse
}

// ExtractEntities - User OpenAI to generate a JSON of Entity Extracts based on the prompt
func (c *OpenAI) ExtractEntities(i int, paraText *string) (ExtractEntitiesResponse, error) {
	var err error
	var resp openai.ChatCompletionResponse
	var eer ExtractEntitiesResponse
	// var ee string

	oaiConfig := openai.DefaultAzureConfig(c.OAIapiKey, c.OAIapibase)

	oaiConfig.AzureModelMapperFunc = func(model string) string {
		azureModelMapping := map[string]string{
			// Map Format OpenAI Model "gpt-3.5-turbo": "Your "gpt-3.5-turbo" deployment name",
			"gpt-3.5-turbo":    "chat",
			"gpt-35-turbo-16k": "gpt-35-turbo-16k",
			"gpt-4":            "gpt-4-8k",
			"gpt-4-32k":        "gpt-4-32k",
			"gpt-4o":           "gpt-4o",
		}
		return azureModelMapping[model]
	}

	// TODO - Add timeout

	// timeout := time.Duration(3 * 60 * time.Second) // 10 seconds
	// Create a custom HTTP client with the specified timeout
	// httpClient := &http.Client{
	// 	Timeout: timeout,
	// }

	client := openai.NewClientWithConfig(oaiConfig)

	cleanText, err := StringCleaner(*paraText)
	if err != nil {
		return eer, fmt.Errorf("ExtractEntities - Paragraph %v - StringClearer error: %v", i, err)
	}

	req2 := openai.ChatCompletionRequest{
		// Ignore the model as it is set in the config
		// Model:       openai.GPT432K,
		Model:       c.OAImodel,
		Temperature: c.OAItemperature,
		// Only proivide one response
		N: 1,
		// MaxTokens: 14000,
		MaxTokens: c.OAImaxTokens,
		Messages: []openai.ChatCompletionMessage{
			// {
			// 	Role:    openai.ChatMessageRoleAssistant,
			// 	Content: c.OIAprompt + "\n\n###\n\n" + cleanText,
			// },

			// Structured Prompt
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: c.OIAprompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: cleanText,
			},
		},
	}

	resp, err = client.CreateChatCompletion(context.Background(), req2)

	// if debug is set display the number of tokens used
	if c.Debug {
		log.Printf("ExtractEntities - DEBUG - Paragraph:%v\nOpenAIConfig:%v\n", i, utils.PrettyStructDebug(req2))
		log.Printf("ExtractEntities - DEBUG - Paragraph:%v\nText:%v\n", i, cleanText)
		log.Printf("ExtractEntities - DEBUG - Paragraph:%v - PromptTokens=%v\n", i, resp.Usage.PromptTokens)
		log.Printf("ExtractEntities - DEBUG - Paragraph:%v - CompletionTokens=%v\n", i, resp.Usage.CompletionTokens)
		log.Printf("ExtractEntities - DEBUG - Paragraph:%v - Total Tokens=%v\n", i, resp.Usage.TotalTokens)
		log.Printf("ExtractEntities - DEBUG - Paragraph:%v - MaxTokens=%v\n", i, c.OAImaxTokens)
	}

	// Now Handle the error
	if err != nil {
		eer.Info = storeChatResponseInfo(&resp)
		return eer, fmt.Errorf("ExtractEntities - Error - Paragraph %v - OpenAI Completion error: %v", i, err)
	}

	// Debug and Completion output handling
	if len(resp.Choices) > 0 && c.Debug {
		fmt.Printf("ExtractEntities - Paragraph %v - Completion Response=%v\n", i, resp.Choices[0].Message.Content)
	} else if len(resp.Choices) == 0 {
		eer.Info = storeChatResponseInfo(&resp)
		return eer, fmt.Errorf("ExtractEntities - Paragraph %v - OpenAI Completion No Value Returned: %v", i, err)
	}

	// Set the value if the processing is ok
	eer.extractEntitiesResponse = resp.Choices[0].Message.Content
	eer.Info = storeChatResponseInfo(&resp)

	return eer, err

}

// storeChatResponseInfo - store the token usage information less the answer
func storeChatResponseInfo(r *openai.ChatCompletionResponse) openai.ChatCompletionResponse {
	u := *r
	u.Choices = nil
	return u
}

// AnalysisResultCount - how many results are there
func (ca *ContentAnalysisData) AnalysisResultCount() int {
	cnt := len(ca.AnalysisResults)
	return cnt
}

// AnalysisErrorCount - how many errors are there
func (ca *ContentAnalysisData) AnalysisErrorCount() int {
	cnt := len(ca.AnalysisErrors)
	return cnt
}

// AnalysisResultError - return the error for the result
func (ca *ContentAnalysisData) AnalysisResultData(i int) interface{} {
	return ca.AnalysisResults[i]
}

// AnalysisResultError - return the error for the result
func (ca *ContentAnalysisData) AnalysisResultError(i int) error {
	ar := ca.AnalysisResults[i]
	return ar.AnalysisMetaData.AnalysisError

}

func (oai *OpenAI) WorkerCount() int {
	return oai.OAIparralelRequests
}

func (oai *OpenAI) AnalyserDisabled() bool {
	return oai.OAIdisable
}

func (oai *OpenAI) AnalyserType() string {
	return "OpenAI"
}

func (oai *OpenAI) AnalyserDebug() bool {
	return oai.Debug
}

// Marshall to clean the strings - Not sure why
func StringCleaner(in string) (string, error) {

	// TODO - Add typical string cleaning function
	// Remove any unwanted characters
	out, err := json.Marshal(stringCleaner(in))
	// out := stringCleaner(in)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func stringCleaner(s string) string {
	replaceStr := " "

	cleanerList := []string{"\n", "\"", "\t", "\r", "*", "|", "(", ")", "[", "]", "{", "}", "/", ",", "?", "%", "\u0026", "+", "/", "@", "\u00a0", ":", "<p>", "<li>", "ul", "\u003e", "```json", "```"}

	for _, clnStr := range cleanerList {
		s = strings.ReplaceAll(s, clnStr, replaceStr)

		s = strings.TrimFunc(s, func(r rune) bool {
			return !unicode.IsGraphic(r)
		})
		// fmt.Println("IN-Clean", destDataValue)
	}

	return s
}
