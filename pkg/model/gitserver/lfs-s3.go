package gitserver

import (
	"context"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3ContentManager struct {
	options   *S3ContentManagerOptions
	s3cli     *s3.Client
	s3presign *s3.PresignClient
}

type S3ContentManagerOptions struct {
	URL          string // the s3 url
	Bucket       string // the bucket name lfs objects will be stored in
	Region       string // the region of the bucket
	Credential   aws.Credentials
	LinkExpireIn time.Duration // the upload/download/verify link expire in,0 means never expired
}

func NewS3ContentManager(ctx context.Context, opts *S3ContentManagerOptions) (*S3ContentManager, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{Value: opts.Credential},
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: opts.URL}, nil
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}
	s3cli := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = opts.Region
		o.UsePathStyle = true
	})
	s3presign := s3.NewPresignClient(s3cli)
	return &S3ContentManager{
		s3cli:     s3cli,
		s3presign: s3presign,
		options:   opts,
	}, nil
}

// Upload get upload url and verify url for a given object
func (m *S3ContentManager) Upload(ctx context.Context, dir string, oid string) (*Link, error) {
	object := &s3.PutObjectInput{
		Bucket: aws.String(m.options.Bucket),
		Key:    aws.String(path.Join(dir, oid)),
	}
	expirein := m.options.LinkExpireIn
	presignresult, err := m.s3presign.PresignPutObject(ctx, object, s3.WithPresignExpires(expirein))
	if err != nil {
		return nil, err
	}
	return &Link{
		Href:      presignresult.URL,
		Header:    HeadersMap(presignresult.SignedHeader),
		ExpireIn:  int(expirein.Seconds()),
		ExpiresAt: time.Now().Add(expirein),
	}, nil
}

// Download get download url for a given object
func (m *S3ContentManager) Download(ctx context.Context, dir string, oid string) (*Link, error) {
	object := &s3.GetObjectInput{
		Bucket: aws.String(m.options.Bucket),
		Key:    aws.String(path.Join(dir, oid)),
	}

	expirein := m.options.LinkExpireIn
	presignResult, err := m.s3presign.PresignGetObject(ctx, object, s3.WithPresignExpires(expirein))
	if err != nil {
		return nil, err
	}
	return &Link{
		Href:      presignResult.URL,
		Header:    HeadersMap(presignResult.SignedHeader),
		ExpireIn:  int(expirein.Seconds()),
		ExpiresAt: time.Now().Add(expirein),
	}, nil
}

func (m *S3ContentManager) Verify(ctx context.Context, path string, oid string) (*BatchObject, error) {
	headresult, err := m.s3cli.HeadObject(ctx, &s3.HeadObjectInput{})
	if err != nil {
		return nil, err
	}
	return &BatchObject{
		OID:  oid,
		Size: headresult.ContentLength,
	}, nil
}
