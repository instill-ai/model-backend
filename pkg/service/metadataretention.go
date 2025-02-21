package service

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/x/minio"
)

// MetadataRetentionHandler allows clients to access the object expiration rule
// associated to a namespace. This is used to set the expiration of objects,
// e.g. when uploading the data of a model run. The preferred way to set the
// expiration of an object is by attaching a tag to the object. The MinIO
// client should set the tag-ased expiration rules for the bucket when it is
// initialized.
type MetadataRetentionHandler interface {
	ListExpiryRules() []minio.ExpiryRule
	GetExpiryRuleByNamespace(_ context.Context, namespaceUID uuid.UUID) (minio.ExpiryRule, error)
}

type metadataRetentionHandler struct{}

// NewRetentionHandler is the default implementation of
// MetadataRetentionHandler. It returns the same expiration rule for all
// namespaces.
func NewRetentionHandler() MetadataRetentionHandler {
	return &metadataRetentionHandler{}
}

var (
	defaultExpiryRule = minio.ExpiryRule{
		Tag:            "default-expiry",
		ExpirationDays: 3,
	}
)

func (h *metadataRetentionHandler) ListExpiryRules() []minio.ExpiryRule {
	return []minio.ExpiryRule{defaultExpiryRule}
}

func (h *metadataRetentionHandler) GetExpiryRuleByNamespace(_ context.Context, _ uuid.UUID) (minio.ExpiryRule, error) {
	return defaultExpiryRule, nil
}
