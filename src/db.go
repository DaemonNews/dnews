package dnews

import (
	"database/sql"

	// postgresql
	_ "github.com/lib/pq"
	"github.com/qbit/pgenv"
)

// DBConnect returns a connection to the database
func DBConnect() (*sql.DB, error) {
	var cstr = pgenv.ConnStr{}
	cstr.SetDefaults()

	return sql.Open("postgres", cstr.ToString())
}

// Auth checks a user's username / password for login
func Auth(u string, p string) (*User, error) {
	var user = &User{}
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
	err = db.QueryRow(`select id, created, fname, lname, email, username, (hash = crypt($1, hash)) as authed from users where username = $2`, p, u).Scan(&user.ID, &user.Created, &user.FName, &user.LName, &user.Email, &user.User, &user.Authed)
	if err != nil {
		return nil, err
	}

	defer db.Close()

	return user, nil
}

// GetRawArticle returns the raw markdown for a given article
func GetRawArticle(slug string) (*Article, error) {
	var a = Article{}
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
	err = db.QueryRow(`
SELECT
 body
from articles
where
  slug = $1
`, slug).Scan(&a.Body)
	if err != nil {
		return nil, err
	}

	defer db.Close()

	return &a, nil
}

// GetBugs grabs all the bugs in the db
func GetBugs() (*Bugs, error) {
	var bs = Bugs{}
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`select * from bugs`)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var b = Bug{}
		rows.Scan(&b.ID, &b.Created, &b.Name, &b.Descr, &b.URL)

		bs = append(bs, &b)
	}

	return &bs, nil
}

// GetArticle returns the raw markdown for a given article
func GetArticle(slug string) (*Article, error) {
	var a = Article{}
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
	err = db.QueryRow(`
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

	a.HTML()

	defer db.Close()

	return &a, nil
}

// GetNArticles returns N most recent articles from the DB
func GetNArticles(n int) (Articles, error) {
	var as = Articles{}
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
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
	defer db.Close()

	for rows.Next() {
		var a = Article{}
		err := rows.Scan(&a.ID, &a.Slug, &a.Date, &a.Title, &a.Body, &a.Author.Pubkey, &a.Author.Email, &a.Author.FName, &a.Author.LName, &a.Signature)
		if err != nil {
			return nil, err
		}
		a.Verify(a.Author.Pubkey)
		a.HTML()
		as = append(as, &a)
	}

	return as, nil
}

// SearchArticles uses pg's TS stuff to query all the articles for passed in values
func SearchArticles(query string, limit int) (Articles, error) {
	var as = Articles{}
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
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
  (pubkeys.userid = users.id), to_tsquery($1) q
  WHERE tsv @@ q
  ORDER BY rank DESC
  LIMIT $2) AS foo

;
		`, query, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	defer db.Close()

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
func InsertUser(u User) (*int, error) {
	var id int
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
	err = db.QueryRow(`INSERT INTO users (fname, lname, email, username, hash) values ($1, $2, $3, (select hash($4)))`, u.FName, u.LName, u.Email, u.User, u.Pass).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// InsertArticle takes an Article and inserts it into the db, it will verify the Author exists
// prior to inserting
func InsertArticle(a Article) (*int, error) {
	var id int
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
	err = db.QueryRow(`INSERT INTO articles (title, body, created, live, sig) values ($1, $2, $3, $4, $5) returning id`, a.Title, a.Body, a.Date, a.Live, a.Signature).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
