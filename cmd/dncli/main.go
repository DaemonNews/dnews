package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/qbit/dnews/src"
)

func insertMD(db *sql.DB, a *dnews.Article) (int, error) {
	var id int
	err := db.QueryRow(`INSERT INTO articles (title, body) values ($1, $2) returning id`, a.Title, a.Body).Scan(&id)
	return id, err
}

func main() {
	var mdFile = flag.String("mdfile", "", "Path to markdown file to import.")
	var pub = flag.String("pubkey", "", "Path to public key for signature verification.")
	var sig = flag.String("sig", "", "Path to signature of article.")
	//var htmlOut = flag.Bool("html", false, "Output in HTML")
	var add = flag.Bool("a", false, "Add aticle to DB")
	flag.Parse()

	if *mdFile == "" {
		fmt.Println("please specify file with -mdfile")
		os.Exit(1)
	}

	var a = dnews.Article{}
	a.LoadFromFile(*mdFile)

	if *pub == "" || *sig == "" {
		fmt.Println("Please specify -pubkey and -sig!")
		os.Exit(1)
	}

	p := dnews.LoadFileOrDie(*pub)
	a.Signature = dnews.LoadFileOrDie(*sig)

	ok, err := a.Verify(p)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !*ok {
		fmt.Println("Signature NOT ok!")
		os.Exit(1)
	}

	fmt.Println("Signature OK")

	if *add {
		db, err := dnews.DBConnect()
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Query(`INSERT INTO articles (title, body, created, live) values ($1, $2, $3, true)`, a.Title, a.Body, a.Date)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Added article!")
	}
}
