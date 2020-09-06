package model

import (
	"encoding/json"
	"io"
)

const (
	POST_CONTENT_LIMIT_LEN = 100
)

type Posts []*Post

func (o Posts) ToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	}

	return string(b)
}

func (o Posts) LimitContentLength() {
	for _, post := range o {
		if len(post.Content) > POST_CONTENT_LIMIT_LEN {
			post.Content = post.Content[:POST_CONTENT_LIMIT_LEN]
		}

		if post.Metadata == nil {
			continue
		}

		for _, comment := range post.Metadata.Comments {
			if len(comment.Content) > POST_CONTENT_LIMIT_LEN {
				comment.Content = comment.Content[:POST_CONTENT_LIMIT_LEN]
			}
		}

		best := post.Metadata.BestAnswer
		if best != nil && len(best.Content) > POST_CONTENT_LIMIT_LEN {
			best.Content = best.Content[:POST_CONTENT_LIMIT_LEN]
		}

		parent := post.Metadata.Parent
		if parent != nil && len(parent.Content) > POST_CONTENT_LIMIT_LEN {
			parent.Content = parent.Content[:POST_CONTENT_LIMIT_LEN]
		}
	}
}

func PostsFromJson(data io.Reader) Posts {
	var o Posts
	json.NewDecoder(data).Decode(&o)
	return o
}

type PostsWithCount struct {
	Posts      Posts `json:"posts"`
	TotalCount int64 `json:"total_count"`
}

func (o *PostsWithCount) ToJson() []byte {
	b, _ := json.Marshal(o)
	return b
}

func PostsWithCountFromJson(data io.Reader) *PostsWithCount {
	var o *PostsWithCount
	json.NewDecoder(data).Decode(&o)
	return o
}

type RelatedPostSearchResultsWithCount struct {
	RelatedPostSearchResults []*RelatedPostSearchResult `json:"related_post_search_results"`
	TotalCount               int64                      `json:"total_count"`
}

func (o *RelatedPostSearchResultsWithCount) ToJson() []byte {
	b, _ := json.Marshal(o)
	return b
}
