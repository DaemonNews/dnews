package dnews

import (
	"database/sql"
	"fmt"

	// postgresql
	"github.com/lib/pq"
	"github.com/qbit/pgenv"
)

// DBConnect returns a connection to the database
func DBConnect() (*sql.DB, error) {
	var cstr = pgenv.ConnStr{}
	cstr.SetDefaults()

	return sql.Open("postgres", cstr.ToString())
}

// Auth checks a user's username / password for login
func Auth(db *sql.DB, u string, p string) (*User, error) {
	var user = &User{}

	err := db.QueryRow(`select id, created, fname, lname, email, username, (hash = crypt($1, hash)) as authed, admin from users where username = $2`, p, u).Scan(&user.ID, &user.Created, &user.FName, &user.LName, &user.Email, &user.User, &user.Authed, &user.Admin)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetRawArticle returns the raw markdown for a given article
func GetRawArticle(db *sql.DB, slug string) (*Article, error) {
	var a = Article{}
	err := db.QueryRow(`
SELECT
 body
from articles
where
  slug = $1
`, slug).Scan(&a.Body)
	if err != nil {
		return nil, err
	}

	return &a, nil
}

// GetBugs grabs all the bugs in the db
func GetBugs(db *sql.DB) (*Bugs, error) {
	var bs = Bugs{}
	rows, err := db.Query(`select * from bugs`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var b = Bug{}
		rows.Scan(&b.ID, &b.Created, &b.Name, &b.Descr, &b.URL)

		bs = append(bs, &b)
	}

	return &bs, nil
}

// GetArticle returns the raw markdown for a given article
func GetArticle(db *sql.DB, slug string) (*Article, error) {
	var a = Article{}
	err := db.QueryRow(`
SELECT
 articles.id,
 slug,
 published,
 title,
 body,
 key,
 email,
 fname,
 lname,
 sig
from articles
join users on
  (articles.authorid = users.id)
join pubkeys on
  (pubkeys.userid = users.id)
where
  articles.slug = $1
`, slug).Scan(&a.ID, &a.Slug, &a.Date, &a.Title, &a.Body, &a.Author.Pubkey, &a.Author.Email, &a.Author.FName, &a.Author.LName, &a.Signature)
	if err != nil {
		return nil, err
	}

	t, err := GetTags(db, a.ID)
	if err != nil {
		return nil, err
	}

	a.Tags = t
	a.HTML()

	return &a, nil
}

// GetTagIDS takes a list of tag names and returns a set of tag ids
func GetTagIDS(db *sql.DB, s []string) (tagIDS []int, err error) {
	sql := `
select
  id
from tags
where
  name = ANY($1)
`
	rows, err := db.Query(sql, pq.Array(s))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var i int
		rows.Scan(&i)
		tagIDS = append(tagIDS, i)
	}

	return tagIDS, nil
}

// GetAllTags returns all the tags in the DB
func GetAllTags(db *sql.DB) (Tags, error) {
	var ts = Tags{}
	rows, err := db.Query(`select id, created, name from tags`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var t = Tag{}
		err := rows.Scan(&t.ID, &t.Created, &t.Name)
		if err != nil {
			return nil, err
		}
		ts = append(ts, &t)
	}

	return ts, nil
}

// GetAllUsers gets all the users in the DB
func GetAllUsers(db *sql.DB) (Users, error) {
	var us = Users{}

	rows, err := db.Query(`select id, created, fname, lname, email, username, admin from users`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var u = User{}
		err := rows.Scan(&u.ID, &u.Created, &u.FName, &u.LName, &u.Email, &u.User, &u.Admin)
		if err != nil {
			return nil, err
		}
		us = append(us, &u)
	}

	return us, nil
}

// GetTags returns tags for a given article
func GetTags(db *sql.DB, id int) (Tags, error) {
	var ts = Tags{}
	rows, err := db.Query(`
		select
		tags.id,
		tags.name
		from article_tags
		join tags on
		(article_tags.tagid = tags.id)
		where
		articleid = $1
		`, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var t = Tag{}
		err := rows.Scan(&t.ID, &t.Name)
		if err != nil {
			return nil, err
		}
		ts = append(ts, &t)
	}
	return ts, nil
}

// GetUserIDByEmail takes a users email and returns a User that is associated with it
func GetUserIDByEmail(db *sql.DB, e string) (id *int, err error) {
	err = db.QueryRow(`
select id from users where email = $1
`, e).Scan(&id)
	if err != nil {
		return nil, err
	}

	return id, nil
}

// GetArticlesByTag tags a tag and returns all the matching articles
func GetArticlesByTag(db *sql.DB, t string) (Articles, error) {
	var as = Articles{}
	rows, err := db.Query(`
		SELECT
		articles.id,
		slug,
		published,
		title,
		body,
		key,
		email,
		fname,
		lname,
		sig
		from articles
		join users on
		(articles.authorid = users.id)
		join pubkeys on
		(pubkeys.userid = users.id)
		join article_tags on
		(article_tags.articleid = articles.id)
		join tags on
		(article_tags.tagid = tags.id)
		where
		live = true and
		tags.name = $1
		order by published desc
		`, t)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var a = Article{}
		err := rows.Scan(&a.ID, &a.Slug, &a.Date, &a.Title, &a.Body, &a.Author.Pubkey, &a.Author.Email, &a.Author.FName, &a.Author.LName, &a.Signature)
		if err != nil {
			return nil, err
		}
		t, err := GetTags(db, a.ID)
		if err != nil {
			return nil, err
		}
		a.Tags = t
		a.Verify(a.Author.Pubkey)
		a.HTML()
		as = append(as, &a)
	}

	return as, nil
}

// GetNArticles returns N most recent articles from the DB
func GetNArticles(db *sql.DB, n int) (Articles, error) {
	var as = Articles{}

	rows, err := db.Query(`
		SELECT
		articles.id,
		slug,
		published,
		title,
		body,
		key,
		email,
		fname,
		lname,
		sig
		from articles
		join users on
		(articles.authorid = users.id)
		join pubkeys on
		(pubkeys.userid = users.id)
		where
		live = true
		order by published desc
		limit $1
		`, n)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var a = Article{}
		err := rows.Scan(&a.ID, &a.Slug, &a.Date, &a.Title, &a.Body, &a.Author.Pubkey, &a.Author.Email, &a.Author.FName, &a.Author.LName, &a.Signature)
		if err != nil {
			return nil, err
		}
		t, err := GetTags(db, a.ID)
		if err != nil {
			return nil, err
		}
		a.Tags = t
		a.Verify(a.Author.Pubkey)
		a.HTML()
		as = append(as, &a)
	}

	return as, nil
}

// SearchArticles uses pg's TS stuff to query all the articles for passed in values
func SearchArticles(db *sql.DB, query string, limit int) (Articles, error) {
	var as = Articles{}
	rows, err := db.Query(`
		SELECT
		aid as id,
		slug,
		published,
		title,
		body,
		key,
		email,
		fname,
		lname,
		sig,
		ts_headline(body, q) as headline,
		rank
		FROM (
			SELECT
			*,
			articles.id as aid,
			ts_rank_cd(tsv, q) as rank
			FROM articles
			join users on
			(articles.authorid = users.id)
			join pubkeys on
			(pubkeys.userid = users.id), plainto_tsquery($1) q
			WHERE tsv @@ q
			ORDER BY rank DESC
			LIMIT $2) AS foo;
		`, query, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var a = Article{}
		err := rows.Scan(&a.ID, &a.Slug, &a.Date, &a.Title, &a.Body, &a.Author.Pubkey, &a.Author.Email, &a.Author.FName, &a.Author.LName, &a.Signature, &a.Headline, &a.Rank)
		if err != nil {
			return nil, err
		}

		a.Verify(a.Author.Pubkey)
		a.HTML()

		as = append(as, &a)
	}

	return as, nil
}

// InsertUser takes a User and inserts them into the database
func InsertUser(db *sql.DB, u User) (*int, error) {
	var id int
	err := db.QueryRow(`INSERT INTO users (fname, lname, email, username, hash) values ($1, $2, $3, (select hash($4)))`, u.FName, u.LName, u.Email, u.User, u.Pass).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// InsertArticle takes an Article and inserts it into the db, it will verify the Author exists
// prior to inserting
func InsertArticle(db *sql.DB, a Article) (*int, error) {
	var id int
	uid, err := AssignUser(db, a.Author.Email)
	if err != nil {
		return nil, err
	}

	a.AuthorID = *uid

	fmt.Printf("AuthorID: %d\n", a.AuthorID)

	err = db.QueryRow(`INSERT INTO articles (title, body, created, live, sig, authorid) values ($1, $2, $3, $4, $5, $6) returning id`, a.Title, a.Body, a.Date, a.Live, a.Signature, a.AuthorID).Scan(&id)
	if err != nil {
		return nil, err
	}

	a.ID = id

	tags, err := GetTagIDS(db, a.Tags.Join())
	if err != nil {
		return nil, err
	}

	err = AssignTags(db, tags, a.ID)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

// AssignUser takes a article (with user already assigned), gets the ID of said user from the db, and creates
// the association assuming the user exists in the db.
func AssignUser(db *sql.DB, e string) (*int, error) {
	uid, err := GetUserIDByEmail(db, e)
	if err != nil {
		return nil, err
	}

	return uid, nil
}

// AssignTags takes a set of tag ids and an article id and creates the association in the article_tags table
func AssignTags(db *sql.DB, ts []int, id int) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("article_tags", "articleid", "tagid"))
	if err != nil {
		return err
	}

	for _, tid := range ts {
		_, err = stmt.Exec(id, tid)
		if err != nil {
			return err
		}
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}
