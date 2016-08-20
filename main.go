package main

import (
	"encoding/gob"
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

func init() {
	gob.Register(&dnews.User{})
}

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
			u, err := dnews.Auth(user, passwd)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if u.Authed {
				session.Values["user"] = u
				session.Save(r, w)
				http.Redirect(w, r, "/", http.StatusFound)
			} else {
				log.Printf("Invalid user: %s", user)
				err = templ.ExecuteTemplate(w, "login.html", struct {
					Error string
				}{
					"Invalid User!",
				})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session.Options = &sessions.Options{
			MaxAge: -1,
		}
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusFound)

	})
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "NO!", http.StatusNotFound)
		return
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

		val := session.Values["user"]
		if _, ok := val.(*dnews.User); !ok {
			val = &dnews.User{}
			session.Values["user"] = &val
			session.Save(r, w)
		}

		data := struct {
			Articles *dnews.Articles
			// we did a type check above, but it would be nice to
			// be able to use the actual type when sending to the
			// templating stuffs :(
			User interface{}
		}{
			&a,
			&val,
		}

		err = templ.ExecuteTemplate(w, "index.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	http.ListenAndServe(":8080", nil)
}
