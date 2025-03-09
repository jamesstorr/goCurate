package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

var ErrNotSupportFormat = errors.New("the file is not supported")

type Document struct {
	XMLName xml.Name `xml:"document"`
	Body    struct {
		P []struct {
			R []struct {
				T struct {
					Text  string `xml:",chardata"`
					Space string `xml:"space,attr"`
				} `xml:"t"`
			} `xml:"r"`
		} `xml:"p"`
	} `xml:"body"`
}

type Paragraph struct {
	R []struct {
		T struct {
			Text  string `xml:",chardata"`
			Space string `xml:"space,attr"`
		} `xml:"t"`
	} `xml:"r"`
}

type Reader struct {
	docxPath *[]byte
	fromDoc  bool
	// docx     *zip.ReadCloser
	docx *zip.Reader
	xml  io.ReadCloser
	dec  *xml.Decoder
}

// NewReader generetes a Reader struct.
// After reading, the Reader struct shall be Close().
func NewReader(docxLoc *[]byte) (*Reader, error) {
	r := new(Reader)
	r.docxPath = docxLoc

	a, err := zip.NewReader(bytes.NewReader((*docxLoc)), int64(len((*docxLoc))))
	if err != nil {
		return nil, err
	}

	// *zip.ReadCloser
	// aa, err := zip.OpenReader(r.docxPath)
	// if err != nil {
	// 	return nil, err
	// }

	var rc io.ReadCloser
	for _, f := range a.File {
		// fmt.Printf("zip file name:%v\n", f.Name)
		// Some docx files have a documentw.xml files NO IDEA !
		if f.Name == "word/document.xml" || f.Name == "word/document2.xml" {
			rc, err = f.Open()
			if err != nil {
				return nil, err
			}
			break
		}
	}

	r.docx = a
	r.xml = rc
	r.dec = xml.NewDecoder(rc)

	return r, nil

}

// Read reads the .docx file by a paragraph.
// When no paragraphs are remained to read, io.EOF error is returned.
func (r *Reader) Read() (string, error) {
	err := seekNextTag(r.dec, "p")
	if err != nil {
		return "", err
	}
	p, err := seekParagraph(r.dec)
	if err != nil {
		return "", err
	}
	return p, nil
}

// ReadAll reads the whole .docx file.
func (r *Reader) ReadAll() ([]string, error) {
	ps := []string{}
	for {
		p, err := r.Read()
		if err == io.EOF {
			return ps, nil
		} else if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
}

func (r *Reader) Close() error {
	r.xml.Close()

	// Shouln't need this close as it's a []byte
	// r.docx.Close()
	// if r.fromDoc {
	// 	os.Remove(r.docxPath)
	// }
	return nil
}

func seekParagraph(dec *xml.Decoder) (string, error) {
	var t string
	for {
		token, err := dec.Token()
		if err != nil {
			return "", err
		}
		switch tt := token.(type) {
		case xml.EndElement:
			if tt.Name.Local == "p" {
				return t, nil
			}
		case xml.StartElement:
			if tt.Name.Local == "t" {
				text, err := seekText(dec)
				if err != nil {
					return "", err
				}
				t = t + text
			}
		}
	}
}

func seekText(dec *xml.Decoder) (string, error) {
	for {
		token, err := dec.Token()
		if err != nil {
			return "", err
		}
		switch tt := token.(type) {
		case xml.CharData:
			return string(tt), nil
		case xml.EndElement:
			return "", nil
		}
	}
}

func seekNextTag(dec *xml.Decoder, tag string) error {
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		t, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if t.Name.Local != tag {
			continue
		}
		break
	}
	return nil
}
