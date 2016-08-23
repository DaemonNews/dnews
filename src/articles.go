package dnews

import (
	//	"database/sql"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ebfe/signify"
	// postgresql
	_ "github.com/lib/pq"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// AuthorRE is a regex to grab our Authors
var AuthorRE = regexp.MustCompile(`^author:\s(.*)$`)

// TitleRE matches our article title
var TitleRE = regexp.MustCompile(`^title:\s(.*)$`)

// DateRE matches our article date
var DateRE = regexp.MustCompile(`^date:\s(.*)$`)

// Article is the base type for all articles
type Article struct {
	ID        int
	Title     string
	Date      time.Time
	Body      []byte
	Author    User
	Signed    bool
	Signature []byte
	Headline  string
	Rank      int
	Tags      Tags
}

// Verify validates the signature of an article
func (a *Article) Verify(pub []byte) (*bool, error) {
	_, pcontent, err := signify.ReadFile(bytes.NewReader(pub))
	_, scontent, err := signify.ReadFile(bytes.NewReader([]byte(a.Signature)))

	if err != nil {
		return nil, err
	}
	sig, err := signify.ParseSignature(scontent)
	if err != nil {
		return nil, err
	}

	pkey, err := signify.ParsePublicKey(pcontent)
	if err != nil {
		return nil, err
	}

	a.Signed = signify.Verify(pkey, a.Body, sig)
	return &a.Signed, nil
}

// LoadFromFile takes the File of a given page and loads the markdown for rendering
func (a *Article) LoadFromFile(p string) error {
	file, err := os.Open(p)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	if err != nil {
		return err
	}

	for scanner.Scan() {
		var line = scanner.Bytes()
		if AuthorRE.Match(line) {
			aline := AuthorRE.ReplaceAllString(string(line), "$1")
			a.Author.Parse(aline)
			fmt.Printf("Author: %s %s (%s)\n", a.Author.FName, a.Author.LName, a.Author.Email)
		}
		if TitleRE.Match(line) {
			a.Title = TitleRE.ReplaceAllString(string(line), "$1")
			fmt.Printf("Title: %s\n", a.Title)
		}
		if DateRE.Match(line) {
			d := DateRE.ReplaceAllString(string(line), "$1")
			a.Date, _ = time.Parse(time.RFC1123, d)
			fmt.Printf("Date: %s\n", a.Date)
		}

		a.Body = append(a.Body, line...)
		a.Body = append(a.Body, 10)
	}

	if err != nil {
		return err
	}
	return nil
}

// Sanatize the htmls
func (a *Article) Sanatize() {
	a.Body = bluemonday.UGCPolicy().SanitizeBytes(a.Body)
}

// HTML returns converted MD to HTML
func (a *Article) HTML() {
	a.Body = blackfriday.MarkdownCommon(a.Body)
	a.Sanatize()
}

// Articles represent a collection of a set of Article
type Articles []*Article

//func (a *Articles) GetSynopsis
