package dnews

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// LoadFileOrDie takes a string and loads a file, returning
func LoadFileOrDie(s string) []byte {
	data, err := ioutil.ReadFile(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return data
}

// FormatDate produces a nicely formatted date
func FormatDate(t time.Time) string {
	return t.Format(time.RFC1123)
}

// ShortDate produces are shorter formatted date
func ShortDate(t time.Time) string {
	return t.Format(time.RFC822)
}
