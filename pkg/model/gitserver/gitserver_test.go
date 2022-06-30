package gitserver

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func ExampleServer_Run() {
	ctx := context.Background()
	url := "http://s3.example.com"
	s3opts := &S3ContentManagerOptions{
		URL:    url,
		Bucket: "git-lfs",
		Credential: aws.Credentials{
			AccessKeyID:     "",
			SecretAccessKey: "",
		},
		LinkExpireIn: time.Hour,
	}
	s3lfs, err := NewS3ContentManager(ctx, s3opts)
	if err != nil {
		panic(err)
	}
	opts := NewDefaultOptions()
	opts.UseGitHTTPBackend = true

	s := Server{
		GitBase: "/var/www/git",
		LFS:     s3lfs,
	}
	if err := s.Run(ctx, opts); err != nil {
		panic(err)
	}
}
