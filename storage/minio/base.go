package storage

import (
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/ravielze/oculi/storage"
)

type (
	impl struct {
		cl      *minio.Client
		signer  *minio.Client
		region  string
		mu      sync.RWMutex
		buckets map[string]storage.Bucket
	}

	bucket struct {
		cl        *minio.Client
		signer    *minio.Client
		name      string
		parent    *impl
		isDeleted bool
		mu        sync.RWMutex
	}
)
