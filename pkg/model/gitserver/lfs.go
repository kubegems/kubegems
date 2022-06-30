package gitserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	OperationUpload   = "upload"
	OperationDownload = "download"
)

// nolint: tagliatelle
type Batch struct {
	// Should be download or upload.
	Operation string `json:"operation,omitempty"`

	//  An optional Array of String identifiers for transfer adapters that the client has configured.
	//  If omitted, the basic transfer adapter MUST be assumed by the server.
	Transfers []string `json:"transfers,omitempty"`

	// String identifier of the transfer adapter that the server prefers.
	// This MUST be one of the given transfer identifiers from the request.
	// Servers can assume the basic transfer adapter if none were given.
	// The Git LFS client will use the basic transfer adapter if the transfer property is omitted.
	Transfer string `json:"transfer,omitempty"`

	// Optional object describing the server ref that the objects belong to. Note: Added in v2.4.
	Ref BatchRef `json:"ref,omitempty"`

	// An Array of objects to download/upload.
	Objects []BatchObject `json:"objects,omitempty"`

	// The hash algorithm used to name Git LFS objects for this repository. Optional;
	// defaults to sha256 if not specified.
	HashAlgo string `json:"hash_algo,omitempty"`
}

type BatchObject struct {
	// String OID of the LFS object.
	OID string `json:"oid,omitempty"`

	// Integer byte size of the LFS object. Must be at least zero.
	Size int64 `json:"size,omitempty"`

	// Optional boolean specifying whether the request for this specific object is authenticated.
	// If omitted or false, Git LFS will attempt to find credentials for this URL.
	// (https://github.com/git-lfs/git-lfs/blob/main/docs/api/authentication.md)
	Authenticated bool `json:"authenticated,omitempty"`

	// Object containing the next actions for this object.
	// Applicable actions depend on which operation is specified in the request.
	// How these properties are interpreted depends on which transfer adapter the client will be using.
	Actions map[string]Link `json:"actions,omitempty"`

	// Describing error if this object.
	Error *ObjectError `json:"error,omitempty"`
}

// LFS servers can respond with these other HTTP status codes:
//
// 401 - The authentication credentials are needed, but were not sent. Git LFS will attempt to get the authentication for the request and retry immediately.
// 403 - The user has read, but not write access. Only applicable when the operation in the request is "upload."
// 404 - The Repository does not exist for the user.
// 422 - Validation error with one or more of the objects in the request. This means that none of the requested objects to upload are valid.
//
// nolint: tagliatelle
type BatchError struct {
	// String error message.
	Message string `json:"message"`

	// Optional String unique identifier for the request. Useful for debugging.
	RequestID string `json:"request_id"`

	// Optional String to give the user a place to report errors.
	DocumentURL string `json:"document_url"`
}

type BatchRef struct {
	// Fully-qualified server refspec.
	Name string `json:"name"`
}

type ObjectError struct {
	// LFS object error codes should match HTTP status codes where possible:
	// 404 - The object does not exist on the server.
	// 409 - The specified hash algorithm disagrees with the server's acceptable options.
	// 410 - The object was removed by the owner.
	// 422 - Validation error.
	Code int `json:"code,omitempty"`
	// String error message.
	Message string `json:"message,omitempty"`
}

// nolint: tagliatelle
type Link struct {
	// String URL to download/upload the object.
	Href string `json:"href,omitempty"`

	// Optional hash of String HTTP header key/value pairs to apply to the request.
	Header map[string]string `json:"header,omitempty"`

	// Whole number of seconds after local client time when transfer will expire.
	// Preferred over expires_at if both are provided. Maximum of 2147483647, minimum of -2147483647.
	ExpireIn int `json:"expire_in,omitempty"`

	// String uppercase RFC 3339-formatted timestamp with second precision
	// for when the given action expires (usually due to a temporary token).
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// LFSBatch takes a batch of operations and returns a response with the
// status of each operation.
// https://github.com/git-lfs/git-lfs/blob/main/docs/api/batch.md#git-lfs-batch-api
func (s *Server) LFSBatch(w http.ResponseWriter, r *http.Request) {
	batch := &Batch{}
	if err := json.NewDecoder(r.Body).Decode(batch); err != nil {
		BadRequest(w, ObjectError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	repopath := s.RepositoryPath(r)
	batchResponse := &Batch{
		Transfer: "basic",
		Objects:  make([]BatchObject, 0, len(batch.Objects)),
		HashAlgo: batch.HashAlgo,
	}

	ctx := r.Context()
	defer r.Body.Close()
	switch batch.Operation {
	case OperationUpload:
		for _, obj := range batch.Objects {
			if link, err := s.LFS.Upload(ctx, repopath, obj.OID); err != nil {
				obj.Error = &ObjectError{Code: http.StatusInternalServerError, Message: err.Error()}
			} else {
				obj.Actions = map[string]Link{
					"upload": *link,
				}
			}
			batchResponse.Objects = append(batchResponse.Objects, obj)
		}
		OK(w, batchResponse)
	case OperationDownload:
		for _, obj := range batch.Objects {
			if link, err := s.LFS.Download(ctx, repopath, obj.OID); err != nil {
				obj.Error = &ObjectError{Code: http.StatusInternalServerError, Message: err.Error()}
			} else {
				obj.Actions = map[string]Link{
					"download": *link,
				}
			}
			batchResponse.Objects = append(batchResponse.Objects, obj)
		}
		OK(w, batchResponse)
	default:
		BadRequest(w, BatchError{
			RequestID:   "",
			DocumentURL: "",
			Message:     fmt.Sprintf("Invalid operation: %s", batch.Operation),
		})
	}
}

func (s *Server) LFSUpload(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (s *Server) LFSDownload(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (s *Server) LFSUpdate(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (s *Server) LFSDelete(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (s *Server) LFSVerify(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

type LFSMetaManager interface {
	// Upload get upload url and verify url for a given object
	Upload(ctx context.Context, path string, oid string) (*Link, error)
	// Download get download url for a given object
	Download(ctx context.Context, path string, oid string) (*Link, error)
	// Verify verfiy object exists
	Verify(ctx context.Context, path string, oid string) (*BatchObject, error)
}
