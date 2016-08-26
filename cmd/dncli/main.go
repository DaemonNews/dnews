package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/DaemonNews/dnews/src"
)

func main() {
	var mdFile = flag.String("mdfile", "", "Path to markdown file to import.")
	var pub = flag.String("pubkey", "", "Path to public key for signature verification.")
	var sig = flag.String("sig", "", "Path to signature of article.")
	//var htmlOut = flag.Bool("html", false, "Output in HTML")
	var add = flag.Bool("a", false, "Add aticle to DB")
	var live = flag.Bool("l", false, "Set article to be live")
	flag.Parse()

	db, err := dnews.DBConnect()

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
	a.Signed = *ok
	a.Live = *live

	if *add {
		id, err := dnews.InsertArticle(db, a)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Added article! (%d)\n", *id)
	}
}
