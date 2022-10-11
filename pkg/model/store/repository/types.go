// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import "time"

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
	ID               string            `json:"id,omitempty" bson:"_id,omitempty"`
	Source           string            `json:"source"` // source of model (huggingface, ...)
	Name             string            `json:"name"`
	Tags             []string          `json:"tags"`
	Author           string            `json:"author"`
	License          string            `json:"license"`
	Framework        string            `json:"framework"`
	Paper            map[string]string `json:"paper"`
	Downloads        int               `json:"downloads"`
	Task             string            `json:"task"`
	Likes            int               `json:"likes"`
	Versions         []ModelVersion    `json:"versions"`                         // versions of model
	Recomment        int               `json:"recomment"`                        // number of recomment votes
	RecommentContent string            `json:"recommentContent"`                 // content of recomment
	CreateAt         *time.Time        `json:"createAt" bson:"create_at"`        // original creation time
	UpdateAt         *time.Time        `json:"updateAt" bson:"update_at"`        // original updattion time
	LastModified     *time.Time        `json:"lastModified" bson:"lastModified"` // last time the model synced
	Enabled          bool              `json:"enabled"`                          // is model published
	Annotations      map[string]string `json:"annotations"`                      // additional infomations
}

type ModelFile struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

const (
	SourceKindHuggingface = "huggingface"
	SourceKindOpenMMLab   = "openmmlab"
	SourceKindModelx      = "modelx"
)

type Source struct {
	// nolint: tagliatelle
	ID           string            `json:"id,omitempty" bson:"_id,omitempty"`
	Name         string            `json:"name"`
	BuiltIn      bool              `json:"builtIn"`
	Online       bool              `json:"online"`
	Images       []string          `json:"images"`
	CreationTime time.Time         `json:"creationTime"`
	UpdationTime time.Time         `json:"updationTime"`
	Enabled      bool              `json:"enabled"`
	InitImage    string            `json:"initImage"` // storage initialize image
	Kind         string            `json:"kind"`      // kind of source (huggingface, openmmlab, modelx...)
	Address      string            `json:"address"`   // address of source
	Auth         SourceAuth        `json:"auth"`      // auth of source
	Annotations  map[string]string `json:"annotations"`
}

type SourceAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type ModelVersion struct {
	Name         string      `json:"name"`
	Files        []ModelFile `json:"files"`
	Intro        string      `json:"intro"`
	CreationTime time.Time   `json:"creationTime"` // original creation time
	UpdationTime time.Time   `json:"updationTime"` // original updattion time
}
