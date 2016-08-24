package dnews

import (
	"time"
)

// Bug is the structure of a BSD User Group
type Bug struct {
	ID      int
	Created time.Time
	Name    string
	Descr   string
	URL     string
}

// Bugs are a collection of bug!
type Bugs []*Bug
