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
func GetRawArticle(id int) (*Article, error) {
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
  id = $1
`, id).Scan(&a.Body)
	if err != nil {
		return nil, err
	}

	defer db.Close()

	return &a, nil
}

// GetArticle returns the raw markdown for a given article
func GetArticle(id int) (*Article, error) {
	var a = Article{}
	db, err := DBConnect()
	if err != nil {
		return nil, err
	}
	err = db.QueryRow(`
SELECT
 articles.id,
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
  articles.id = $1
`, id).Scan(&a.ID, &a.Date, &a.Title, &a.Body, &a.Author.Pubkey, &a.Author.Email, &a.Author.FName, &a.Author.LName, &a.Signature)
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
		err := rows.Scan(&a.ID, &a.Date, &a.Title, &a.Body, &a.Author.Pubkey, &a.Author.Email, &a.Author.FName, &a.Author.LName, &a.Signature)
		if err != nil {
			return nil, err
		}
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
	rows, err := db.Query(`SELECT id, published, title, ts_headline(body, q) as headline, rank
		FROM (SELECT id, published, title, q, ts_rank_cd(tsv, q) AS rank
			FROM articles, to_tsquery($1) q
			WHERE tsv @@ q
			ORDER BY rank DESC
			LIMIT $2) AS foo;
		`, query, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	defer db.Close()

	for rows.Next() {
		var a = Article{}
		err := rows.Scan(&a.ID, &a.Date, &a.Title, &a.Headline, &a.Author, &a.Rank)
		if err != nil {
			return nil, err
		}
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
	err = db.QueryRow(`INSERT INTO articles (title, body, created, live) values ($1, $2, $3, true)`, a.Title, a.Body, a.Date).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
