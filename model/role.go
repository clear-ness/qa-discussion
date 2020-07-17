package model

type Role struct {
	Name        string
	Permissions []string
}

var ROLE_NORMAL *Role
var ROLE_MODERATOR *Role
var ROLE_ADMIN *Role

var ALL_ROLES []*Role

func initializeRoles() {
	ROLE_NORMAL = &Role{
		Name: USER_TYPE_NORMAL,
		Permissions: []string{
			PERMISSION_CREATE_POST.Id,
			PERMISSION_EDIT_POST.Id,
			PERMISSION_DELETE_POST.Id,
			PERMISSION_DELETE_USER.Id,
			PERMISSION_VOTE_POST.Id,
			PERMISSION_FLAG_POST.Id,
			PERMISSION_FAVORITE_POST.Id,
		},
	}

	ROLE_MODERATOR = &Role{
		Name: USER_TYPE_MODERATOR,
		Permissions: append(
			[]string{
				PERMISSION_DELETE_OTHERS_POSTS.Id,
				PERMISSION_DELETE_OTHER_USERS.Id,
				PERMISSION_LOCK_POST.Id,
				PERMISSION_PROTECT_POST.Id,
				PERMISSION_SUSPEND_USER.Id,
				PERMISSION_READ_OTHERS_TAGS.Id,
			},
			ROLE_NORMAL.Permissions...,
		),
	}

	ROLE_ADMIN = &Role{
		Name: USER_TYPE_ADMIN,
		Permissions: append(
			[]string{
				PERMISSION_EDIT_OTHERS_POSTS.Id,
				PERMISSION_EDIT_OTHER_USERS.Id,
				PERMISSION_EDIT_OTHER_USERS_PASSWORD.Id,
				PERMISSION_EDIT_USER_TYPE.Id,
				PERMISSION_READ_OTHERS_INBOX_MESSAGES.Id,
				PERMISSION_SET_READ_OTHERS_INBOX_MESSAGES.Id,
				PERMISSION_READ_OTHERS_USER_POINT_HISTORY.Id,
				PERMISSION_READ_OTHERS_VOTES.Id,
			},
			ROLE_MODERATOR.Permissions...,
		),
	}

	ALL_ROLES = []*Role{
		ROLE_NORMAL,
		ROLE_MODERATOR,
		ROLE_ADMIN,
	}
}

func init() {
	initializeRoles()
}
