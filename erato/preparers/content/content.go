package content

import (
	"Erato/erato/preparers/docx"
	"bytes"
	"fmt"
	"strings"

	pdf "github.com/dslipak/pdf"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

// Config - Configuration for the Content Preparer
type Config struct {
	ParagraphMaxWordCount int
	ParagraphMinWordCount int
	Debug                 bool
}

// Type of Content that the Prepare function will convert to text
type DOCX struct{ Config }
type HTML struct{ Config }
type PDF struct{ Config }

// TODO - .pdf file extension will convert from pdf to text
// TODO - .html file extension will convert from html to text

// Prepare - function that converts a Word document file to text of string format
func (dt DOCX) Prepare(docData *[]byte) ([]string, error) {

	c := dt.Config
	var err error
	var chunks []string

	// fp := filepath.Clean(doc.FileName)
	r, err := docx.NewReader(docData)
	if err != nil {
		return chunks, fmt.Errorf("convertWordToText-%v", err)
	}
	defer r.Close()

	// Read all the paragraphs
	// Need to change this to a reader of byte slices

	paraGraphs, err := r.ReadAll()
	if err != nil {
		return chunks, fmt.Errorf("convertWordToText-%v", err)
	}

	// Read all the paragraphs
	for _, paraGraph := range paraGraphs {

		// Process the paragraph
		pts := strings.Fields(paraGraph)
		// less than 5 words in a paragraph then ignore
		if len(pts) <= c.ParagraphMinWordCount {
			continue
		}

		// chunk further if bigger than the chunk size
		if len(pts) > c.ParagraphMaxWordCount {
			// If it is then create smaller chunks of text to then add onto the chunks []string
			chunks = append(chunks, chunkyVator(c.ParagraphMaxWordCount, pts)...)

		} else {
			chunks = append(chunks, strings.Join(pts, " "))
		}

	}

	// Return the chunked text in a string slice
	return chunks, err
}

// Prepare - Takes HTML date and converts to text using the openAPI service
func (dt HTML) Prepare(docData *[]byte) ([]string, error) {
	var err error
	c := dt.Config
	// debug := c.Debug

	converter := md.NewConverter("", true, nil)

	markdown, err := converter.ConvertString(string(*docData))

	// if debug {
	// 	fmt.Println("--------------------------------------------------------------------")
	// 	fmt.Println(markdown)
	// 	fmt.Println("--------------------------------------------------------------------")
	// }

	// Split into a slice of words
	pts := strings.Fields(markdown)

	chunks := chunkyVator(c.ParagraphMaxWordCount, pts)

	return chunks, err

}

// Prepare - Takes HTML date and converts to text using the openAPI service
func (dt PDF) Prepare(docData *[]byte) ([]string, error) {
	var err error
	c := dt.Config
	var buf bytes.Buffer

	//  create a new reader from the docData byte slice
	reader := bytes.NewReader(*docData)

	// convert size to and in64
	size := int64(len(*docData))

	// Create a new pdf reader
	pdfr, err := pdf.NewReader(reader, size)
	if err != nil {
		return nil, err
	}

	io, err := pdfr.GetPlainText()
	if err != nil {
		return nil, err
	}

	// Read the io.Reader into the buffer and convert to a string
	buf.ReadFrom(io)
	content := buf.String()

	// Split into a slice of words
	pts := strings.Fields(content)

	chunks := chunkyVator(c.ParagraphMaxWordCount, pts)

	return chunks, err

}

// Return a []string of chunks of text - each chunk is of length c.ChunkWordCount
func chunkyVator(wc int, pts []string) []string {
	// create a []string of chunks of words of greater than or equal to c.ChunkWordCount
	var chunks []string
	for i := 0; i < len(pts); i += wc {

		// Determine the value of end and ensure its goes beyond the end of the wordSlice
		var end int
		if i+wc <= len(pts) {
			end = i + wc
		} else {
			end = len(pts)
		}

		chunk := pts[i:end]
		chunks = append(chunks, strings.Join(chunk, " "))
	}

	return chunks

}
