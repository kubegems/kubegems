package mongo

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Options struct {
	Addr     string `json:"addr,omitempty" description:"mongodb address"`
	Database string `json:"database,omitempty" description:"mongodb database"`
	Username string `json:"username,omitempty" description:"mongodb username"`
	Password string `json:"password,omitempty" description:"mongodb password"`
}

func DefaultOptions() *Options {
	return &Options{
		Addr:     "mongo:27017",
		Database: "models",
		Username: "",
		Password: "",
	}
}

func NewMongoDB(ctx context.Context, opt *Options) (*mongo.Client, *mongo.Database, error) {
	mongoopt := &options.ClientOptions{
		Hosts: strings.Split(opt.Addr, ","),
	}
	if opt.Username != "" && opt.Password != "" {
		mongoopt.Auth = &options.Credential{
			Username: opt.Username,
			Password: opt.Password,
		}
	}
	mongocli, err := mongo.NewClient(mongoopt)
	if err != nil {
		return nil, nil, err
	}
	if err := mongocli.Connect(ctx); err != nil {
		return nil, nil, err
	}
	if err := mongocli.Ping(ctx, nil); err != nil {
		return nil, nil, err
	}
	return mongocli, mongocli.Database(opt.Database), nil
}
