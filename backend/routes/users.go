package routes

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"main/sql"
	"main/utils"
	"mime/multipart"
	"net/http"
)

// IsActiveRoute is a middleware function that checks if the user is active
func IsActiveRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var response struct {
			IsOnline  bool   `json:"isOnline"`
			SessionId string `json:"sessionId"`
		}
		err := json.NewDecoder(r.Body).Decode(&response)
		if err != nil {
			return
		}

		var user *sql.User

		if response.SessionId != "" {
			user, err = sql.GetUserBySession(response.SessionId)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			user, err = LoginUser(r)
			if err != nil {
				log.Fatal(err)
			}
		}

		if user == nil {
			return
		}

		err = sql.SetUserOnline(user.Id, response.IsOnline)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// SettingsRoute is a route that handles the settings of the user
func SettingsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if TemplatesData.ConnectedUser == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		err := utils.CallTemplate("settings", TemplatesData, w)
		if err != nil {
			log.Fatal(err)
		}
	}

	if r.Method == "POST" {
		// max size = 100 Mb
		err := r.ParseMultipartForm(100 << 20)
		if err != nil {
			log.Fatal(err)
		}

		newUser := sql.User{
			Username:    r.FormValue("username"),
			Email:       r.FormValue("email"),
			Description: r.FormValue("description"),
			Locale:      r.FormValue("locale"),
		}

		var profilePicture string
		file, header, err := r.FormFile("profile-picture")
		if header != nil {
			defer func(file multipart.File) {
				err := file.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(file)
		}
		if err != nil {
			if err != http.ErrMissingFile {
				log.Fatal(err)
			}
		} else {
			profilePicture = "data:" + header.Header.Get("Content-Type") + ";base64,"

			buf := bytes.NewBuffer(nil)
			if _, err := io.Copy(buf, file); err != nil {
				log.Fatal(err)
			}

			profilePicture += base64.StdEncoding.EncodeToString(buf.Bytes())
			newUser.ProfilePic = profilePicture
		}

		user, err := LoginUser(r)
		if err != nil {
			log.Fatal(err)
		}

		_, err = sql.EditUser(user.Id, newUser)
		if err != nil {
			log.Fatal(err)
		}

		TemplatesData.ConnectedUser = sql.GetUserById(user.Id)

		r.Method = "GET"
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// UsersActive is a middleware function that checks if the users sent are active
func UsersActive(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		if r.Header.Get("Content-Type") != "application/json" {
			return
		}

		var response struct {
			Users []string `json:"users"`
		}
		err := json.NewDecoder(r.Body).Decode(&response)
		if err != nil {
			log.Fatal(err)
		}

		usersOnline, err := sql.GetUsersStatus(response.Users)
		if err != nil {
			log.Fatal(err)
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(usersOnline)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func UsersOfflineRoute() {
	sql.SetUsersOffline()
}
