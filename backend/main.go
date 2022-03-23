package main

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	. "main/sql"
	"net/http"
	"strings"
)

type TemplatesDataType struct {
	ConnectedUser *User
}

var TemplatesData = TemplatesDataType{}

func main() {
	templates, err := template.New("").ParseGlob("../client/templates/*.gohtml")
	if err != nil {
		log.Fatal(err)
	}

	css := http.FileServer(http.Dir("../client/style/"))       //define css file
	http.Handle("/static/", http.StripPrefix("/static/", css)) //set css file to static

	resources := http.FileServer(http.Dir("../backend/resources/"))        //define css file
	http.Handle("/resources/", http.StripPrefix("/resources/", resources)) //set css file to static

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("route / request")
		path := strings.TrimPrefix(r.URL.Path, "/")
		cookie, err := r.Cookie("session")
		if err != nil {
			fmt.Println("not logged in")
			TemplatesData.ConnectedUser = nil
		} else {
			fmt.Println("logged in")
			user, err := UserLoginBySession(cookie.Value)
			if err != nil {
				log.Fatal(err)
			}
			TemplatesData.ConnectedUser = user
		}

		if path == "favicon.ico" {
			return
		}
		if path == "" {
			fmt.Println("index page loaded")
			fmt.Println(TemplatesData)
			fmt.Println(TemplatesData.ConnectedUser)
			err := templates.ExecuteTemplate(w, "index.gohtml", TemplatesData)
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Println("login page loaded")
			err := templates.ExecuteTemplate(w, "login.gohtml", nil)
			if err != nil {
				log.Fatal(err)
			}
		}

		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				log.Fatal(err)
			}

			username := r.FormValue("username")
			exists, err := UserLogin(
				username,
				r.FormValue("password"),
			)
			if err != nil {
				log.Fatal(err)
			}

			if exists {
				user := GetUserByUsername(username)
				SessionID(*user, w)
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
		}
	})

	http.HandleFunc("/sign-up", func(w http.ResponseWriter, r *http.Request) {
		err := templates.ExecuteTemplate(w, "sign-up.gohtml", nil)
		if err != nil {
			log.Fatal(err)
		}

		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				log.Fatal(err)
			}

			err = SaveUser(NewUser(
				r.FormValue("username"),
				r.FormValue("password"),
				r.FormValue("email"),
			))
			if err != nil {
				log.Fatal(err)
			}
		}

	})

	http.HandleFunc("/admin/edit-username", func(w http.ResponseWriter, r *http.Request) {
		err := templates.ExecuteTemplate(w, "admin_edit_username.gohtml", nil)
		if err != nil {
			log.Fatal(err)
		}

		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				log.Fatal(err)
			}

			err = AdminEditUsername(
				r.FormValue("old-username"),
				r.FormValue("new-username"),
			)

			if err != nil {
				log.Fatal(err)
			}
			return
		}
	})

	http.HandleFunc("/edit-username", func(w http.ResponseWriter, r *http.Request) {
		err := templates.ExecuteTemplate(w, "edit_username.gohtml", nil)
		if err != nil {
			log.Fatal(err)
		}

		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				log.Fatal(err)
			}
			//get cookie from browser
			cookie, err := r.Cookie("session")
			if err != nil {
				log.Fatal(err)
			}

			//select user from session
			result, err := DB.Query("SELECT id_user FROM sessions WHERE id_session = ?", cookie.Value)
			if err != nil {
				log.Fatal(err)
			}

			//close select query
			defer func(result *sql.Rows) {
				err := result.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(result)

			//get result from query
			var idUser int64
			if result.Next() {
				err = result.Scan(&idUser)
			}

			userConnected := GetUserById(idUser)
			TemplatesData.ConnectedUser = userConnected

			//edit username of idUser
			err = EditUsername(
				idUser,
				r.FormValue("new-username"),
			)

			if err != nil {
				log.Fatal(err)
			}
			return
		}
	})

	// Capture connection properties.
	cfg := mysql.Config{
		User:                 "root",
		Passwd:               "",
		Net:                  "tcp",
		Addr:                 "",
		DBName:               "forum",
		AllowNativePasswords: true,
		ParseTime:            true,
	}
	// Get a database handle.
	DB, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := DB.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	fmt.Println("Server started at localhost:8091")
	err = http.ListenAndServe(":8091", nil)
	if err != nil {
		log.Fatal(err)
	}

}
