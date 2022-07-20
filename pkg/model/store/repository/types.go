package repository

import "time"

/*
post:										<--- post id

[post content]

comments:

- [user a] at 00:12 : nice job!				<--- comment(id)
- [user b] at 00:15 : where is the cat?		<--- comment's content
- [user c] at 00:35 : > where is the cat?	<--- reply to a comment(id)
					  in the dark!
*/
type Comment struct {
	// nolint: tagliatelle
	ID           string    `json:"id,omitempty" bson:"_id,omitempty"` // id
	Username     string    `json:"username"`                          // user's id (username)
	PostID       string    `json:"postID"`                            // post id
	Content      string    `json:"content"`                           // comment's content
	ReplyTo      *ReplyTo  `json:"replyTo" `                          // comment's content(some section) reply to
	Rating       int       `json:"rating"`                            // rating value (1-5)
	CreationTime time.Time `json:"creationTime"`                      // comment's create time
	UpdationTime time.Time `json:"updationTime"`                      // comment's update time
}

type ReplyTo struct {
	// nolint: tagliatelle
	ID           string    `json:"id,omitempty" bson:"_id,omitempty"` // id of the comment that this comment reply to
	RootID       string    `json:"rootID"`                            // id of the top level comment, if this comment is a reply to a reply
	Content      string    `json:"content"`                           // comment's content
	Username     string    `json:"username"`                          // user's id (username)
	CreationTime time.Time `json:"creationTime"`                      // comment's create time
}

// model meta in models store
// nolint: tagliatelle
type Model struct {
	ID           string                 `json:"id,omitempty" bson:"_id,omitempty"`
	Source       string                 `json:"source"` // source of model (huggingface, ...)
	Name         string                 `json:"name"`
	Registry     string                 `json:"registry"`
	Type         string                 `json:"type"`
	Tags         []string               `json:"tags"`
	Author       string                 `json:"author"`
	License      string                 `json:"license"`
	Files        []ModelFile            `json:"files"`
	Framework    string                 `json:"framework"`
	Paper        map[string]string      `json:"paper"`
	Intro        string                 `json:"intro"`
	Downloads    int                    `json:"downloads"`
	Task         string                 `json:"task"`
	Likes        int                    `json:"likes"`
	Raw          map[string]interface{} `json:"raw"`
	Recomment    int                    `json:"recomment"` // number of recomment votes
	CreateAt     *time.Time             `json:"createAt" bson:"create_at"`
	UpdateAt     *time.Time             `json:"updateAt" bson:"update_at"`
	LastModified *time.Time             `json:"lastModified" bson:"lastModified"`
}

type ModelFile struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type Source struct {
	// nolint: tagliatelle
	ID           string    `json:"id,omitempty" bson:"_id,omitempty"`
	Name         string    `json:"name"`
	Desc         string    `json:"desc"`
	Icon         string    `json:"icon"`
	BuiltIn      bool      `json:"builtIn"`
	Enabled      bool      `json:"enabled"`
	Images       []string  `json:"images"`
	CreationTime time.Time `json:"creationTime"`
	UpdationTime time.Time `json:"updationTime"`
}
