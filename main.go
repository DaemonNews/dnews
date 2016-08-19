package main

import (
	"fmt"
	"html/template"
	"net/http"

	// Gimme that sweet sweet pg

	"github.com/qbit/dnews/src"
)

func getArticles(w http.ResponseWriter, r *http.Request) {

}

func handler(w http.ResponseWriter, r *http.Request) {
	a := dnews.Articles{}
	fmt.Fprintf(w, "Hi there, I love %s! %v", r.URL.Path[1:], a)
}

var funcMap = template.FuncMap{
	"formatDate": dnews.FormatDate,
	"printHTML": func(b []byte) template.HTML {
		return template.HTML(string(b))
	},
}

var idxT, err = template.New("dnews").Funcs(funcMap).ParseGlob("templates/*.html")

func main() {
	fs := http.FileServer(http.Dir("public"))

	http.Handle("/public/", http.StripPrefix("/public/", fs))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		a, err := dnews.GetNArticles(10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = idxT.ExecuteTemplate(w, "index.html", a)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	http.ListenAndServe(":8080", nil)
}
