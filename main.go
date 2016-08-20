package main

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
	r := mux.NewRouter()
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))
	r.HandleFunc("/article/raw/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		article, err := dnews.GetRawArticle(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", article.Body)
	})
	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
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
	r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
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
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "NO!", http.StatusNotFound)
		return
	})
	r.HandleFunc("/advocacy", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
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
			User interface{}
		}{
			&val,
		}

		err = templ.ExecuteTemplate(w, "advocacy.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
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

		a, err := dnews.GetNArticles(10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
