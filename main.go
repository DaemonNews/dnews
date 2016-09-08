package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DaemonNews/dnews/src"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/csrf"
	"github.com/gorilla/feeds"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var insecure bool
var cookieSecret string
var crsfSecret string
var jwtSecret string
var templ *template.Template
var store *sessions.CookieStore
var listen string

type response struct {
	Error string
	User  interface{}
	Data  interface{}
	CSRF  map[string]interface{}
}

var funcMap = template.FuncMap{
	"formatDate": dnews.FormatDate,
	"shortDate":  dnews.ShortDate,
	"printByte": func(b []byte) string {
		return string(b)
	},
	"joinTags": func(ts dnews.Tags) template.HTML {
		var s []string
		for _, t := range ts {

			s = append(s, fmt.Sprintf(`<a href="/tag/%s">%s</a>`, t.Name, t.Name))
		}
		return template.HTML(strings.Join(s, ", "))
	},
	"printHTML": func(b []byte) template.HTML {
		return template.HTML(string(b))
	},
}

func init() {
	var err error
	flag.BoolVar(&insecure, "i", false, "Insecure mode")
	flag.StringVar(&cookieSecret, "cookie", "something-very-secret", "Secret to use for cookie store")
	flag.StringVar(&crsfSecret, "crsf", "32-byte-long-auth-key", "Secret to use for cookie store")
	flag.StringVar(&jwtSecret, "jwt", "super secret neat", "Secret to use for jwt")
	flag.StringVar(&listen, "http", ":8080", "Listen on")

	flag.Parse()

	store = sessions.NewCookieStore([]byte(cookieSecret))
	templ, err = template.New("dnews").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal(err)
	}

	gob.Register(&dnews.User{})
}

func renderTemplate(w http.ResponseWriter, r *http.Request, d *response, t string) {
	d.CSRF = map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
	}
	err := templ.ExecuteTemplate(w, t, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func grabUser(w http.ResponseWriter, r *http.Request) (*response, error) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		return nil, err
	}

	uVal := session.Values["user"]

	var _, ok = uVal.(*dnews.User)

	if !ok {
		uVal = &dnews.User{}
		session.Values["user"] = &uVal
		session.Save(r, w)
	}

	var data = response{}
	data.User = &uVal
	return &data, nil
}

func main() {
	db, err := dnews.DBConnect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := mux.NewRouter()
	router.PathPrefix("/public/").Handler(
		http.StripPrefix("/public/",
			http.FileServer(http.Dir("public"))))

	router.HandleFunc("/feeds", func(w http.ResponseWriter, r *http.Request) {
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		renderTemplate(w, r, data, "feeds.html")
	})
	router.HandleFunc("/ml", func(w http.ResponseWriter, r *http.Request) {
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		renderTemplate(w, r, data, "ml.html")
	})
	router.HandleFunc("/archives", func(w http.ResponseWriter, r *http.Request) {
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		renderTemplate(w, r, data, "archives.html")
	})
	router.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.FormValue("search")
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Can't get user: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		a, err := dnews.SearchArticles(db, query, 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data.Data = &a

		renderTemplate(w, r, data, "search_results.html")
	})

	router.HandleFunc("/feed/{type}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		feedType := vars["type"]
		now := time.Now()
		feed := &feeds.Feed{
			Title:       "Daemon.News",
			Link:        &feeds.Link{Href: "https://daemon.news"},
			Description: "*BSD News and Advocacy",
			Author:      &feeds.Author{Name: "The Daemon News Team", Email: "daemons@daemon.news"},
			Created:     now,
			Copyright:   "This work is copyright © Daemon.News",
		}

		a, err := dnews.GetNArticles(db, 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// TODO populate body?
		feed.Items = []*feeds.Item{}

		for _, article := range a {
			f := feeds.Item{}
			f.Title = article.Title
			f.Link = &feeds.Link{Href: fmt.Sprintf("http://daemon.news/article/%s", article.Slug)}
			f.Author = &feeds.Author{Name: article.Author.FName, Email: article.Author.Email}
			f.Created = article.Date

			feed.Items = append(feed.Items, &f)
		}

		switch feedType {
		case "atom":
			atom, err := feed.ToAtom()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, atom)
		case "rss":
			rss, err := feed.ToRss()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, rss)
		default:
			http.Error(w, "Invalid feed type!", http.StatusInternalServerError)
			return
		}

	})
	router.HandleFunc("/tag/{tag:[a-zA-Z0-9-]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tag := vars["tag"]

		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		articles, err := dnews.GetArticlesByTag(db, tag)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data.Data = articles
		renderTemplate(w, r, data, "index.html")

	})
	router.HandleFunc("/article/{slug:[a-zA-Z0-9-]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slug := vars["slug"]

		article, err := dnews.GetArticle(db, slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Data = article
		renderTemplate(w, r, data, "article.html")

	})
	router.HandleFunc("/article/raw/{slug:[a-zA-Z0-9-]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slug := vars["slug"]

		article, err := dnews.GetRawArticle(db, slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", article.Body)
	})
	router.HandleFunc("/login/post", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user := r.FormValue("user")
		passwd := r.FormValue("passwd")

		if user == "" && passwd == "" {
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			u, err := dnews.Auth(db, user, passwd)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if u.Authed {
				session.Values["user"] = u
				session.Save(r, w)
				http.Redirect(w, r, "/", http.StatusFound)
			}
		}
	})
	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		renderTemplate(w, r, data, "login.html")
	})
	router.HandleFunc("/api/{type}/{action}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		typ := vars["type"]
		action := vars["action"]

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Not Authorized!", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(authHeader, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if token.Valid {
			switch typ {
			default:
				http.Error(w, "Invalid API Requests!", http.StatusNotImplemented)
				return
			case "status":
				if action == "ok" {
					fmt.Fprint(w, "OK")
				}

			}
		}
	})

	router.HandleFunc("/api/gentoken", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		uVal := session.Values["user"]
		var u, ok = uVal.(*dnews.User)
		if !ok {
			uVal = &dnews.User{}
			session.Values["user"] = &uVal
			session.Save(r, w)
		}

		var data = &response{}
		data.User = &u

		if ok && u.Admin {
			// Check our token field even if we haven't set it before
			token, err := jwt.Parse(u.Token, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil {
				if ve, ok := err.(*jwt.ValidationError); ok {
					if ve.Errors&jwt.ValidationErrorMalformed != 0 {
						data.Error = "Invalid token"
					} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
						data.Error = "Previous token expired"
					} else {
						data.Error = err.Error()
					}
				}
			}

			if !token.Valid {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"eml": u.Email,
					// 168 = 7 days
					"exp": time.Now().Add(time.Hour * 168).Unix(),
					"nbf": time.Now().Unix(),
				})

				tokenString, err := token.SignedString([]byte(jwtSecret))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				u.Token = tokenString
				session.Values["user"] = &u
				session.Save(r, w)
			}
			data.Data = u.Token
			renderTemplate(w, r, data, "token.html")
		}
	})

	router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		uVal := session.Values["user"]
		var u, ok = uVal.(*dnews.User)
		if !ok {
			uVal = &dnews.User{}
			session.Values["user"] = &uVal
			session.Save(r, w)
		}

		var data = &response{}
		data.User = &u

		if ok {
			if u.Admin {
				t, err := dnews.GetAllTags(db)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				us, err := dnews.GetAllUsers(db)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				data.Data = struct {
					*dnews.Tags
					*dnews.Users
				}{
					&t,
					&us,
				}

				renderTemplate(w, r, data, "admin.html")
			} else {
				renderTemplate(w, r, data, "perm_denied.html")
			}
		} else {
			renderTemplate(w, r, data, "login.html")
		}

	})
	router.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
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
	router.HandleFunc("/advocacy", func(w http.ResponseWriter, r *http.Request) {
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		bugs, err := dnews.GetBugs(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data.Data = &bugs

		renderTemplate(w, r, data, "advocacy.html")
	})
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := grabUser(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		a, err := dnews.GetNArticles(db, 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data.Data = &a
		renderTemplate(w, r, data, "index.html")
	})

	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

	if insecure {
		log.Fatal(http.ListenAndServe(listen,
			csrf.Protect([]byte("32-byte-long-auth-key"),
				csrf.Secure(false))(loggedRouter)))
	} else {
		log.Fatal(http.ListenAndServe(listen,
			csrf.Protect([]byte(crsfSecret))(loggedRouter)))

	}
}
