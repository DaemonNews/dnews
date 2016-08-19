package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/qbit/dnews/src"
)

var store = sessions.NewCookieStore([]byte("something-very-secret"))

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

var templ, err = template.New("dnews").Funcs(funcMap).ParseGlob("templates/*.html")

func main() {
	fs := http.FileServer(http.Dir("public"))

	http.Handle("/public/", http.StripPrefix("/public/", fs))
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		user := r.FormValue("user")
		passwd := r.FormValue("passwd")

		if user == "" && passwd == "" {
			err = templ.ExecuteTemplate(w, "login.html", nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// do auth
			log.Printf("Authed %s", user)
			session.Values["user"] = dnews.User{}
			fmt.Printf("%v", session.Values["user"])
			//session.Values["user"].Authed = true

			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		a, err := dnews.GetNArticles(10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var u = dnews.User{}
		u.Authed = true
		u.FName = "sucka"
		fmt.Println(u)

		data := struct {
			Articles *dnews.Articles
		}{
			&a,
		}
		session.Save(r, w)
		err = templ.ExecuteTemplate(w, "header.html", u)
		err = templ.ExecuteTemplate(w, "index.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	http.ListenAndServe(":8080", nil)
}
