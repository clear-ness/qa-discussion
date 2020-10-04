package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) SetPostMetadata(posts model.Posts, options model.SetPostMetadataOptions) (model.Posts, *model.AppError) {
	var postIds []string
	var parentIds []string
	userIdsMaps := map[string]bool{}
	for _, post := range posts {
		postIds = append(postIds, post.Id)
		userIdsMaps[post.UserId] = true
		if post.ParentId != "" {
			parentIds = append(parentIds, post.ParentId)
		}
	}

	userMap := map[string]*model.User{}
	// TODO: チームに紐づく投稿の場合は、ユーザー情報はteamに関するものにする
	// (teamMember.points, teamMember.typeなど)
	if options.SetUser {
		var userIds []string
		for key := range userIdsMaps {
			userIds = append(userIds, key)
		}

		users, err := a.GetUsers(userIds)
		if err != nil {
			return nil, err
		}

		sanitizeOptions := map[string]bool{}
		sanitizeOptions["email"] = false
		for _, user := range users {
			user.SanitizeProfile(sanitizeOptions)
			userMap[user.Id] = user
		}
	}

	var comments map[string][]*model.Post
	if options.SetComments {
		var err *model.AppError
		comments, err = a.GetCommentsAndCommentUserForPosts(postIds)
		if err != nil {
			return nil, err
		}
	}

	parentMap := map[string]*model.Post{}
	if options.SetParent && len(parentIds) > 0 {
		parentIdsMaps := map[string]bool{}
		for _, parentId := range parentIds {
			parentIdsMaps[parentId] = true
		}

		var parentIds []string
		for key := range parentIdsMaps {
			parentIds = append(parentIds, key)
		}

		parents, err := a.Srv.Store.Post().GetPostsByIds(parentIds)
		if err != nil {
			return nil, err
		}

		for _, parent := range parents {
			parentMap[parent.Id] = parent
		}
	}

	for _, post := range posts {
		post.Metadata = &model.PostMetadata{}

		if options.SetUser {
			if user, ok := userMap[post.UserId]; ok {
				post.Metadata.User = user
			}
		}

		if options.SetComments {
			post.Metadata.Comments = comments[post.Id]
		}

		if options.SetBestAnswer && len(post.BestId) > 0 {
			best, err := a.GetPostWithMetadata(post.BestId)
			if err != nil {
				return nil, err
			}

			post.Metadata.BestAnswer = best
		}

		if options.SetParent {
			if parent, ok := parentMap[post.ParentId]; ok {
				post.Metadata.Parent = parent
			}
		}
	}

	return posts, nil
}
