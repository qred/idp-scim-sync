package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"github.com/slashdevops/idp-scim-sync/internal/model"
)

// Consume s3.Client

var (
	// ErrS3ClientNil is returned when s3 client is nil
	ErrS3ClientNil = errors.New("s3: AWS S3 Client is nil")

	// ErrOptionWithBucketNil is returned when WitBucket option is nil
	ErrOptionWithBucketNil = errors.New("s3: option WithBucket is nil")

	// ErrOptionWithKeyNil is returned when WithKey option is nil
	ErrOptionWithKeyNil = errors.New("s3: option WithKey is nil")

	// ErrStateNil is returned when state is nil
	ErrStateNil = errors.New("s3: state is nil")
)

// S3Repository represent a repository that stores state in S3 and implements model.Repository interface
type S3Repository struct {
	bucket string
	key    string
	client S3ClientAPI
}

// NewS3Repository returns a new S3Repository
func NewS3Repository(client S3ClientAPI, opts ...S3RepositoryOption) (*S3Repository, error) {
	if client == nil {
		return nil, ErrS3ClientNil
	}

	s3r := &S3Repository{
		client: client,
	}

	for _, opt := range opts {
		opt(s3r)
	}

	if s3r.bucket == "" {
		return nil, ErrOptionWithBucketNil
	}

	if s3r.key == "" {
		return nil, ErrOptionWithKeyNil
	}

	return s3r, nil
}

// GetState returns the state from the repository
func (r *S3Repository) GetState(ctx context.Context) (*model.State, error) {
	resp, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(r.key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: error getting S3 object: bucket: %s, error: %w", r.bucket, err)
	}
	defer resp.Body.Close()

	var state model.State
	dec := json.NewDecoder(resp.Body)

	if err = dec.Decode(&state); err != nil {
		return nil, fmt.Errorf("s3: error decoding S3 object: %w", err)
	}

	return &state, nil
}

// SetState sets the state in the given repository
func (r *S3Repository) SetState(ctx context.Context, state *model.State) error {
	if state == nil {
		return ErrStateNil
	}

	jsonPayload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("s3: error marshaling state: %w", err)
	}

	_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(r.key),
		Body:   bytes.NewReader(jsonPayload),
	})
	if err != nil {
		return fmt.Errorf("s3: error putting S3 object: %w", err)
	}

	return nil
}
