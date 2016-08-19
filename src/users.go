package dnews

import (
	"regexp"
	"time"
)

// User represents an author of an article
type User struct {
	ID      int
	Created time.Time
	LName   string
	FName   string
	Email   string
	Pubkey  string
	User    string
	Authed  bool
}

var userLineRE = regexp.MustCompile(`^(.*)\s(.*)\s<(.*)>$`)

// Parse takes a 'First Last <user@email.com>' style string and creates a User
func (u *User) Parse(s string) {
	u.FName = userLineRE.ReplaceAllString(s, "$1")
	u.LName = userLineRE.ReplaceAllString(s, "$2")
	u.Email = userLineRE.ReplaceAllString(s, "$3")
}

// Users are a collection of User
type Users *[]User
