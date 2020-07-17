package model

type Permission struct {
	Id string
}

var PERMISSION_CREATE_POST *Permission
var PERMISSION_EDIT_POST *Permission
var PERMISSION_EDIT_OTHERS_POSTS *Permission
var PERMISSION_DELETE_POST *Permission
var PERMISSION_DELETE_OTHERS_POSTS *Permission
var PERMISSION_DELETE_USER *Permission
var PERMISSION_DELETE_OTHER_USERS *Permission
var PERMISSION_EDIT_OTHER_USERS *Permission
var PERMISSION_EDIT_OTHER_USERS_PASSWORD *Permission
var PERMISSION_VOTE_POST *Permission
var PERMISSION_FLAG_POST *Permission
var PERMISSION_READ_OTHERS_INBOX_MESSAGES *Permission
var PERMISSION_SET_READ_OTHERS_INBOX_MESSAGES *Permission
var PERMISSION_READ_OTHERS_USER_POINT_HISTORY *Permission
var PERMISSION_READ_OTHERS_VOTES *Permission
var PERMISSION_READ_OTHERS_TAGS *Permission
var PERMISSION_FAVORITE_POST *Permission
var PERMISSION_LOCK_POST *Permission
var PERMISSION_PROTECT_POST *Permission
var PERMISSION_SUSPEND_USER *Permission
var PERMISSION_EDIT_USER_TYPE *Permission

var ALL_PERMISSIONS []*Permission

func initializePermissions() {
	PERMISSION_CREATE_POST = &Permission{
		"create_post",
	}

	PERMISSION_EDIT_POST = &Permission{
		"edit_post",
	}

	PERMISSION_EDIT_OTHERS_POSTS = &Permission{
		"edit_others_posts",
	}

	PERMISSION_DELETE_POST = &Permission{
		"delete_post",
	}

	PERMISSION_DELETE_OTHERS_POSTS = &Permission{
		"delete_others_posts",
	}

	PERMISSION_DELETE_USER = &Permission{
		"delete_user",
	}

	PERMISSION_DELETE_OTHER_USERS = &Permission{
		"delete_other_users",
	}

	PERMISSION_EDIT_OTHER_USERS = &Permission{
		"edit_other_users",
	}

	PERMISSION_EDIT_OTHER_USERS_PASSWORD = &Permission{
		"edit_other_users",
	}

	PERMISSION_VOTE_POST = &Permission{
		"vote_post",
	}

	PERMISSION_FLAG_POST = &Permission{
		"flag_post",
	}

	PERMISSION_READ_OTHERS_INBOX_MESSAGES = &Permission{
		"read_others_inbox_messages",
	}

	PERMISSION_SET_READ_OTHERS_INBOX_MESSAGES = &Permission{
		"set_read_others_inbox_messages",
	}

	PERMISSION_READ_OTHERS_USER_POINT_HISTORY = &Permission{
		"read_others_user_point_history",
	}

	PERMISSION_READ_OTHERS_VOTES = &Permission{
		"read_others_votes",
	}

	PERMISSION_READ_OTHERS_TAGS = &Permission{
		"read_others_tags",
	}

	PERMISSION_FAVORITE_POST = &Permission{
		"favorite_post",
	}

	PERMISSION_LOCK_POST = &Permission{
		"lock_post",
	}

	PERMISSION_PROTECT_POST = &Permission{
		"protect_post",
	}

	PERMISSION_SUSPEND_USER = &Permission{
		"suspend_user",
	}

	PERMISSION_EDIT_USER_TYPE = &Permission{
		"edit_user_type",
	}

	ALL_PERMISSIONS = []*Permission{
		PERMISSION_CREATE_POST,
		PERMISSION_EDIT_POST,
		PERMISSION_EDIT_OTHERS_POSTS,
		PERMISSION_DELETE_POST,
		PERMISSION_DELETE_OTHERS_POSTS,
		PERMISSION_DELETE_USER,
		PERMISSION_DELETE_OTHER_USERS,
		PERMISSION_EDIT_OTHER_USERS,
		PERMISSION_EDIT_OTHER_USERS_PASSWORD,
		PERMISSION_VOTE_POST,
		PERMISSION_FLAG_POST,
		PERMISSION_READ_OTHERS_INBOX_MESSAGES,
		PERMISSION_SET_READ_OTHERS_INBOX_MESSAGES,
		PERMISSION_READ_OTHERS_USER_POINT_HISTORY,
		PERMISSION_READ_OTHERS_VOTES,
		PERMISSION_READ_OTHERS_TAGS,
		PERMISSION_FAVORITE_POST,
		PERMISSION_PROTECT_POST,
		PERMISSION_SUSPEND_USER,
		PERMISSION_EDIT_USER_TYPE,
	}
}

func init() {
	initializePermissions()
}
