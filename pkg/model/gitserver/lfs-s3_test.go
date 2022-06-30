package gitserver

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestS3ContentManager_Upload(t *testing.T) {
	ctx := context.Background()

	m, err := NewS3ContentManager(ctx, &S3ContentManagerOptions{
		URL:    "https://s3.amazonaws.com",
		Bucket: "git-lfs",
		Credential: aws.Credentials{
			AccessKeyID:     "",
			SecretAccessKey: "",
		},
		LinkExpireIn: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.Upload(ctx, "user/repo", "123")
	if err != nil {
		t.Errorf("S3ContentManager.Upload() error = %v", err)
		return
	}
	log.Print(got)
}
