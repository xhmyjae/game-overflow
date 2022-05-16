package routes

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	sql2 "main/sql"
	"main/utils"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

		var user *sql2.User

		if response.SessionId != "" {
			user, err = sql2.GetUserBySession(response.SessionId)
			if err != nil {
				return
			}
		} else {
			user, err = sql2.GetUserByRequest(r)
			if err != nil {
				utils.RouteError(err)
			}
		}

		if user == nil {
			return
		}

		err = sql2.SetUserOnline(user.Id, response.IsOnline)
		if err != nil {
			utils.RouteError(err)
		}
	}
}

// ProfileRoute is a route that returns the user profile
func ProfileRoute(w http.ResponseWriter, r *http.Request) {
	var userId int64
	var user *sql2.User
	var err error

	query := r.URL.Query()
	if query.Has("id") {
		userIdString := query.Get("id")
		userId, err = strconv.ParseInt(userIdString, 10, 64)

		if err == nil {
			user, err = sql2.GetUserById(userId)
		}
	}

	if user == nil {
		user, err = sql2.GetUserByRequest(r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if user != nil {
			query.Del("id")
		}
	}

	if user == nil || user.Username == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	templateData, err := GetTemplatesDataFromRoute(w, r)
	if err != nil {
		utils.RouteError(err)
	}

	templateData.ShownUser = user

	err = utils.CallTemplate("profile", templateData, w)
	if err != nil {
		utils.RouteError(err)
	}
}

// SettingsRoute is a route that handles the settings of the user
func SettingsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templateData, err := GetTemplatesDataFromRoute(w, r)
		if err != nil {
			utils.RouteError(err)
		}
		if templateData.ConnectedUser == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		templateData.ConnectedUser.CalculateDefaultColor()

		err = utils.CallTemplate("settings", templateData, w)
		if err != nil {
			utils.RouteError(err)
		}
	}

	if r.Method == "POST" {
		err := r.ParseMultipartForm(100 << 20)
		if err != nil {
			utils.RouteError(err)
		}

		colorValue := r.FormValue("color")
		colorValue = strings.TrimPrefix(colorValue, "#")
		color, err := strconv.ParseInt(colorValue, 16, 32)
		if err != nil {
			utils.RouteError(err)
		}

		newUser := sql2.User{
			Username:    r.FormValue("username"),
			Email:       r.FormValue("email"),
			Description: sql.NullString{Valid: true, String: string(r.FormValue("description"))},
			Locale:      r.FormValue("locale"),
			Color:       int(color),
		}

		var profilePicture string
		file, header, err := r.FormFile("profile-picture")
		if header != nil {
			defer func(file multipart.File) {
				err := file.Close()
				if err != nil {
					utils.RouteError(err)
				}
			}(file)
		}
		if err != nil {
			if err != http.ErrMissingFile {
				utils.RouteError(err)
			}
		} else {
			profilePicture = "data:" + header.Header.Get("Content-Type") + ";base64,"

			bytes, err := io.ReadAll(file)
			if err != nil {
				utils.RouteError(err)
			}

			profilePicture += base64.StdEncoding.EncodeToString(bytes)
			newUser.ProfilePic.String = profilePicture
		}

		user, err := sql2.GetUserByRequest(r)
		if err != nil {
			utils.RouteError(err)
		}

		_, err = sql2.EditUser(user.Id, newUser)
		if err != nil {
			utils.RouteError(err)
		}

		r.Method = "GET"
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
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
			utils.RouteError(err)
		}

		usersOnline, err := sql2.GetUsersStatus(response.Users)
		if err != nil {
			utils.RouteError(err)
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(usersOnline)
		if err != nil {
			utils.RouteError(err)
		}
	}
}

func UserPostsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templateData, err := GetTemplatesDataFromRoute(w, r)
		if err != nil {
			utils.RouteError(err)
		}

		if templateData.ConnectedUser == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		query := r.URL.Query()
		if query.Has("id") {
			queryId := query.Get("id")
			id, err := strconv.ParseInt(queryId, 10, 64)
			if err != nil {
				utils.RouteError(err)
			}

			user, err := sql2.GetUserById(id)
			if err != nil {
				utils.RouteError(err)
			}

			topics, err := sql2.GetUserTopics(user.Id)

			if err != nil {
				utils.RouteError(err)
			}

			templateData.ShownTopics = topics

			err = utils.CallTemplate("feed", templateData, w)
			if err != nil {
				utils.RouteError(err)
			}
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}

	}
}

func UserLikesRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templateData, err := GetTemplatesDataFromRoute(w, r)
		if err != nil {
			utils.RouteError(err)
		}

		if templateData.ConnectedUser == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		query := r.URL.Query()
		if query.Has("id") {
			queryId := query.Get("id")
			id, err := strconv.ParseInt(queryId, 10, 64)
			if err != nil {
				utils.RouteError(err)
			}

			user, err := sql2.GetUserById(id)
			if err != nil {
				utils.RouteError(err)
			}

			if templateData.ConnectedUser.Id != user.Id {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}

			topics, err := sql2.GetUserLikesTopics(user.Id)
			if err != nil {
				utils.RouteError(err)
			}

			messages, err := sql2.GetUserLikedMessages(user.Id)
			if err != nil {
				utils.RouteError(err)
			}

			templateData.ShownTopics = topics
			templateData.ShownMessages = messages

			err = utils.CallTemplate("feed", templateData, w)
			if err != nil {
				utils.RouteError(err)
			}
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}
}

func UserBan(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()

	if queries.Has("id") {
		idUser := queries.Get("id")
		fmt.Println(idUser)
		Id, err := strconv.ParseInt(idUser, 10, 64)

		user, err := sql2.GetUserByRequest(r)
		if err != nil {
			utils.RouteError(err)
		}

		if user.Role == "admin" {
			err := sql2.BanUser(Id)
			if err != nil {
				utils.RouteError(err)
			}
		}
		queriesId := url.Values{}
		queriesId.Add("id", idUser)

		http.Redirect(w, r, "/profile?"+queriesId.Encode(), http.StatusSeeOther)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func UserUnban(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()

	if queries.Has("id") {
		fmt.Println("unban func")
		idUser := queries.Get("id")
		Id, err := strconv.ParseInt(idUser, 10, 64)

		user, err := sql2.GetUserByRequest(r)
		if err != nil {
			utils.RouteError(err)
		}

		if user.Role == "admin" {
			err := sql2.UnbanUser(Id)
			if err != nil {
				utils.RouteError(err)
			}
		}
		queriesId := url.Values{}
		queriesId.Add("id", idUser)

		http.Redirect(w, r, "/profile?"+queriesId.Encode(), http.StatusSeeOther)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func FollowUserRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		templateData, err := GetTemplatesDataFromRoute(w, r)
		if err != nil {
			utils.RouteError(err)
		}

		if templateData.ConnectedUser == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		body := r.Body
		defer body.Close()

		var response struct {
			Id string `json:"id"`
		}
		err = json.NewDecoder(body).Decode(&response)
		if err != nil {
			utils.RouteError(err)
		}

		idUserFollowed, err := strconv.ParseInt(response.Id, 10, 64)
		if err != nil {
			utils.RouteError(err)
		}

		err = sql2.FollowUser(idUserFollowed, templateData.ConnectedUser.Id)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func UnfollowUserRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		user, err := sql2.GetUserByRequest(r)
		if err != nil || user == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		body := r.Body
		defer body.Close()

		var response struct {
			Id string `json:"id"`
		}
		err = json.NewDecoder(body).Decode(&response)
		if err != nil {
			utils.RouteError(err)
		}
		idUserFollowed, err := strconv.ParseInt(response.Id, 10, 64)
		if err != nil {
			utils.RouteError(err)
		}

		err = sql2.UnfollowUser(idUserFollowed, user.Id)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
