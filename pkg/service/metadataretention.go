package service

import (
	"context"

	"github.com/gofrs/uuid"

	miniox "github.com/instill-ai/x/minio"
)

type MetadataRetentionHandler interface {
	GetExpiryTagBySubscriptionPlan(ctx context.Context, requesterUID uuid.UUID) (string, error)
}

type metadataRetentionHandler struct{}

func NewRetentionHandler() MetadataRetentionHandler {
	return &metadataRetentionHandler{}
}

func (h metadataRetentionHandler) GetExpiryTagBySubscriptionPlan(ctx context.Context, requesterUID uuid.UUID) (string, error) {
	return defaultExpiryTag, nil
}

const (
	defaultExpiryTag = "default-expiry"
)

var MetadataExpiryRules = []miniox.ExpiryRule{
	{
		Tag:            defaultExpiryTag,
		ExpirationDays: 3,
	},
}
