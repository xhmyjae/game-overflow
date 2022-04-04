package routes

import (
	"log"
	"main/sql"
	"main/utils"
	"net/http"
	"strings"
	"time"
)

type TemplatesDataType struct {
	ConnectedUser *sql.User
	Locales       map[string]string
	ShownTopics   []sql.Topic
	ShownTopic    sql.Topic
}

var TemplatesData = TemplatesDataType{
	Locales: map[string]string{"en": "English", "fr": "Français"},
}

var PageLoadedTime time.Time

func IndexRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	cookie, err := r.Cookie("session")

	if err != nil {
		TemplatesData.ConnectedUser = nil
	} else {
		user, err := sql.GetUserBySession(cookie.Value)
		if err != nil {
			log.Fatal(err)
		}
		TemplatesData.ConnectedUser = user
		//go ResetFirstTimer()
	}

	if path == "" {
		err := utils.CallTemplate("main", TemplatesData, w)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func LogHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(".css", r.URL.String()) && !strings.HasSuffix(".png", r.URL.String()) {
		log.Printf("%v %v", r.Method, r.URL.String())
		go func() {
			PageLoadedTime = time.Now()
		}()
	}
	http.DefaultServeMux.ServeHTTP(w, r)
}