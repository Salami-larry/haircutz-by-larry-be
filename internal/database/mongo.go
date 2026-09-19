package database

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Mongo struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func Connect(ctx context.Context, uri string) (*Mongo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer pingCancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	dbName, err := databaseNameFromURI(uri)
	if err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return &Mongo{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

func (m *Mongo) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return m.Client.Ping(ctx, readpref.Primary())
}

func (m *Mongo) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

func databaseNameFromURI(uri string) (string, error) {
	const defaultName = "haircutz"

	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse mongodb uri: %w", err)
	}

	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return defaultName, nil
	}

	return path, nil
}
