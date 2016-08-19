package dnews

import (
	"time"

	"github.com/ebfe/signify"
)

// Comment is the structure respresenting a single comment
type Comment struct {
	Date      time.Time
	UserID    int
	ArticleID int
	UserName  string
	Parent    int
	Signed    bool
	Body      string
}

// Verify sets the Signed value for a given comment
func (c *Comment) Verify(pub *signify.PublicKey, msg []byte, sig *signify.Signature) {
	c.Signed = signify.Verify(pub, msg, sig)
}

// Comments is a collection of one or more comments
type Comments []*Comment
