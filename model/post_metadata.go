package model

type PostMetadata struct {
	// post's creator
	User *User `json:"author,omitempty"`
	// question's or answer's comments
	Comments []*Post `json:"comments,omitempty"`
	// question's best answer
	BestAnswer *Post `json:"best_answer,omitempty"`
	// parent post
	Parent *Post `json:"parent,omitempty"`
}

type SetPostMetadataOptions struct {
	SetUser       bool
	SetComments   bool
	SetBestAnswer bool
	SetParent     bool
}
