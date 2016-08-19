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
	defer rows.Close()

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
