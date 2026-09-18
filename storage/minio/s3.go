package storage

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ravielze/oculi/storage"
)

func New(endpoint, username, password string, useSSL bool) (storage.S3, error) {
	return NewWithPublicEndpoint(endpoint, endpoint, username, password, useSSL, useSSL, "us-east-1")
}

// NewWithPublicEndpoint signs browser URLs with the public host while keeping
// storage operations on the endpoint reachable by the backend.
func NewWithPublicEndpoint(endpoint, publicEndpoint, username, password string, useSSL, publicUseSSL bool, region string) (storage.S3, error) {
	if region == "" {
		region = "us-east-1"
	}
	newClient := func(address string, secure bool) (*minio.Client, error) {
		return minio.New(address, &minio.Options{Creds: credentials.NewStaticV4(username, password, ""), Secure: secure, Region: region, BucketLookup: minio.BucketLookupPath})
	}
	client, err := newClient(endpoint, useSSL)
	if err != nil {
		return nil, err
	}
	if publicEndpoint == "" {
		publicEndpoint = endpoint
		publicUseSSL = useSSL
	}
	signer := client
	if publicEndpoint != endpoint || publicUseSSL != useSSL {
		signer, err = newClient(publicEndpoint, publicUseSSL)
		if err != nil {
			return nil, err
		}
	}
	return &impl{
		cl:      client,
		signer:  signer,
		region:  region,
		buckets: make(map[string]storage.Bucket),
	}, nil
}

func (i *impl) ListBuckets() ([]storage.BucketInfo, error) {
	var result []storage.BucketInfo
	buckets, err := i.cl.ListBuckets(context.Background())
	if err != nil {
		return nil, err
	}
	for _, x := range buckets {
		result = append(result,
			storage.BucketInfo{
				Name:      x.Name,
				CreatedAt: x.CreationDate,
			},
		)
	}
	return result, nil
}

func (i *impl) BucketExists(bucketName string) (bool, error) {
	found, err := i.cl.BucketExists(context.Background(), bucketName)
	if err != nil {
		return false, err
	}
	return found, nil
}

func (i *impl) GetBucket(bucketName string) storage.Bucket {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.buckets[bucketName]
}

func (i *impl) InitBucket(bucketName string) (storage.Bucket, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.buckets[bucketName] != nil {
		return i.buckets[bucketName], nil
	}

	exists, err := i.BucketExists(bucketName)
	if err != nil {
		return nil, err
	}

	if !exists {
		err := i.cl.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: i.region})
		if err != nil {
			return nil, err
		}
	}
	i.buckets[bucketName] = &bucket{
		cl:        i.cl,
		signer:    i.signer,
		name:      bucketName,
		parent:    i,
		isDeleted: false,
	}
	return i.buckets[bucketName], nil
}
