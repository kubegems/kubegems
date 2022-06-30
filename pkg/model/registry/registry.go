package registry

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-logr/logr"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/model/gitserver"
)

type Options struct {
	Listen string       `json:"listen,omitempty" description:"http server listen address"`
	S3     LFSS3Options `json:"s3,omitempty" description:"s3 options"`
	Git    GitOptions   `json:"git,omitempty" description:"git options"`
}

type GitOptions struct {
	Dir string `json:"dir,omitempty"` // base git directory
}

type LFSS3Options struct {
	Addr         string        `json:"addr,omitempty" description:"s3 url"`
	Bucket       string        `json:"bucket,omitempty" description:"s3 bucket name the lfs objects are stored in"`
	AccessKey    string        `json:"accesskey,omitempty" description:"s3 access key"`
	SecretKey    string        `json:"secretkey,omitempty" description:"s3 secret key"`
	LinkExpireIn time.Duration `json:"linkexpirein,omitempty" description:"lfs bacth api returned links expire in"`
}

func DefaultOptions() *Options {
	return &Options{
		Listen: ":8080",
		S3: LFSS3Options{
			Addr:         "http://s3.example.com",
			Bucket:       "git-lfs",
			LinkExpireIn: time.Hour,
		},
		Git: GitOptions{
			Dir: "repositories",
		},
	}
}

func Run(ctx context.Context, opts *Options) error {
	ctx = log.NewContext(ctx, log.LogrLogger)

	s3lfsman, err := gitserver.NewS3ContentManager(ctx, &gitserver.S3ContentManagerOptions{
		URL:    opts.S3.Addr,
		Bucket: opts.S3.Bucket,
		Credential: aws.Credentials{
			AccessKeyID:     opts.S3.AccessKey,
			SecretAccessKey: opts.S3.SecretKey,
		},
		LinkExpireIn: opts.S3.LinkExpireIn,
	})
	if err != nil {
		return err
	}
	s := gitserver.Server{GitBase: opts.Git.Dir, LFS: s3lfsman}
	log := logr.FromContextOrDiscard(ctx)
	log.Info("starting git http server", "listen", opts.Listen)
	if err := s.Run(ctx, &gitserver.Options{Listen: opts.Listen, UseGitHTTPBackend: true}); err != nil {
		return err
	}
	return nil
}
