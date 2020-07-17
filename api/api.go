package api

import (
	"fmt"
	"net/http"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/web"
	"github.com/gorilla/mux"
)

type Routes struct {
	Root    *mux.Router // ''
	ApiRoot *mux.Router // 'api/v1'

	Users *mux.Router // 'api/v1/users'
	User  *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}'

	Tags *mux.Router // 'api/v1/tags'

	Posts        *mux.Router // 'api/v1/posts'
	Post         *mux.Router // 'api/v1/posts/{post_id:[A-Za-z0-9]+}'
	PostsForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/posts'

	Files *mux.Router // 'api/v1/files'
	File  *mux.Router // 'api/v1/files/{file_id:[A-Za-z0-9]+}'

	InboxMessagesForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/inbox_messages'
	InboxMessageForUser  *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/inbox_messages/{inbox_message_id:[A-Za-z0-9]+}'

	UserPointHistoryForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/user_point_history'

	VotesForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/votes'

	UserFavoritePosts *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/user_favorite_posts'

	NotificationSettingForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/notification_setting'
}

type API struct {
	GetGlobalAppOptions app.AppOptionCreator
	BaseRoutes          *Routes
}

func Init(globalOptionsFunc app.AppOptionCreator, root *mux.Router) *API {
	api := &API{
		GetGlobalAppOptions: globalOptionsFunc,
		BaseRoutes:          &Routes{},
	}

	api.BaseRoutes.Root = root
	api.BaseRoutes.ApiRoot = root.PathPrefix(model.API_URL_SUFFIX).Subrouter()

	api.BaseRoutes.Users = api.BaseRoutes.ApiRoot.PathPrefix("/users").Subrouter()
	api.BaseRoutes.User = api.BaseRoutes.ApiRoot.PathPrefix("/users/{user_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.Tags = api.BaseRoutes.ApiRoot.PathPrefix("/tags").Subrouter()

	api.BaseRoutes.Posts = api.BaseRoutes.ApiRoot.PathPrefix("/posts").Subrouter()
	api.BaseRoutes.Post = api.BaseRoutes.Posts.PathPrefix("/{post_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.PostsForUser = api.BaseRoutes.User.PathPrefix("/posts").Subrouter()

	api.BaseRoutes.Files = api.BaseRoutes.ApiRoot.PathPrefix("/files").Subrouter()
	api.BaseRoutes.File = api.BaseRoutes.ApiRoot.PathPrefix("/files/{file_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.InboxMessagesForUser = api.BaseRoutes.User.PathPrefix("/inbox_messages").Subrouter()
	api.BaseRoutes.InboxMessageForUser = api.BaseRoutes.InboxMessagesForUser.PathPrefix("/{inbox_message_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.UserPointHistoryForUser = api.BaseRoutes.User.PathPrefix("/user_point_history").Subrouter()

	api.BaseRoutes.VotesForUser = api.BaseRoutes.User.PathPrefix("/votes").Subrouter()

	api.BaseRoutes.UserFavoritePosts = api.BaseRoutes.User.PathPrefix("/user_favorite_posts").Subrouter()

	api.BaseRoutes.NotificationSettingForUser = api.BaseRoutes.User.PathPrefix("/notification_setting").Subrouter()

	api.InitUser()
	api.InitPost()
	api.InitTag()
	api.InitUserFavoritePost()
	api.InitFile()
	api.InitNotificationSetting()

	root.Handle("/api/v1/{anything:.*}", http.HandlerFunc(hello))

	return api
}

func hello(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "hello")
}

var ReturnStatusOK = web.ReturnStatusOK
