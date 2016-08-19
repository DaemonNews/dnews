package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ebfe/signify"
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
	var verify = flag.Bool("v", false, "Verify signature (requires -pubkey and -sig).")
	var add = flag.Bool("a", false, "Add aticle to DB")
	flag.Parse()

	if *mdFile == "" {
		fmt.Println("please specify file with -mdfile")
		os.Exit(1)
	}

	var a = dnews.Article{}
	a.LoadFromFile(*mdFile)

	if *verify {
		if *pub == "" || *sig == "" {
			fmt.Println("Please specify -pubkey and -sig!")
			os.Exit(1)
		}

		pubdata := dnews.LoadFileOrDie(*pub)
		sigdata := dnews.LoadFileOrDie(*sig)

		pubkey, err := signify.ParsePublicKey(pubdata)
		if err != nil {
			log.Fatal(err)
		}
		signature, err := signify.ParseSignature(sigdata)
		if err != nil {
			log.Fatal(err)
		}
		if a.Verify(pubkey, signature) {
			fmt.Printf("Signature OK")
		} else {
			log.Fatal("Invalid Signature!")
		}
	}

	if *add {
		db, err := dnews.DBConnect()
		if err != nil {
			log.Fatal(err)
		}
		var id int
		err = db.QueryRow(`INSERT INTO articles (title, body, created) values ($1, $2, $3) returning id`, a.Title, a.Body, a.Date).Scan(&id)

		if err != nil {
			log.Fatal(err)
		}
	}
}
