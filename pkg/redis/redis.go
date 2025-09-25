package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	URL          string `split_words:"true" required:"true"`
	ReadTimeout  int    `split_words:"true" default:"3"`
	WriteTimeout int    `split_words:"true" default:"3"`
	DialTimeout  int    `split_words:"true" default:"5"`
}

func (r *Config) New() (*redis.Client, error) {
	opts, err := redis.ParseURL(r.URL)
	if err != nil {
		return nil, err
	}

	opts.ReadTimeout = time.Duration(r.ReadTimeout) * time.Second
	opts.WriteTimeout = time.Duration(r.WriteTimeout) * time.Second
	opts.DialTimeout = time.Duration(r.DialTimeout) * time.Second

	client := redis.NewClient(opts)

	cmd := client.Ping(context.Background())
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	return client, nil
}

func (r *Config) MustNew() *redis.Client {
	client, err := r.New()
	if err != nil {
		panic(err)
	}

	return client
}
