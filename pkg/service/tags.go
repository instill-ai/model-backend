package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	httpclient "github.com/instill-ai/model-backend/pkg/client/http"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	errorsx "github.com/instill-ai/x/errors"
	logx "github.com/instill-ai/x/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DeleteRepositoryTag deletes a repository tag from both the database and the registry
func (s *service) DeleteRepositoryTag(ctx context.Context, req *modelpb.DeleteRepositoryTagRequest) (*modelpb.DeleteRepositoryTagResponse, error) {
	name := utils.RepositoryTagName(req.GetName())
	repo, id, err := name.ExtractRepositoryAndID()
	if err != nil {
		return nil, fmt.Errorf("invalid tag name")
	}

	rt, err := s.repository.GetRepositoryTag(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing tag %s: %w", id, err)
	}

	registryClient := httpclient.NewRegistryClient(ctx, s.cfg.Registry.Host, s.cfg.Registry.Port)
	if err := registryClient.DeleteTag(ctx, repo, rt.Digest); err != nil {
		return nil, err
	}

	if err := s.repository.DeleteRepositoryTag(ctx, rt.Digest); err != nil {
		return nil, err
	}

	return &modelpb.DeleteRepositoryTagResponse{}, nil
}

// CreateRepositoryTag stores the tag information of a pushed repository content.
func (s *service) CreateRepositoryTag(ctx context.Context, req *modelpb.CreateRepositoryTagRequest) (*modelpb.CreateRepositoryTagResponse, error) {
	name := utils.RepositoryTagName(req.GetTag().GetName())
	_, id, err := name.ExtractRepositoryAndID()
	if err != nil || id != req.GetTag().GetId() {
		return nil, fmt.Errorf("invalid tag name")
	}

	// Convert protobuf to domain model
	tag := &datamodel.Tag{
		Name:   req.GetTag().GetName(),
		ID:     req.GetTag().GetId(),
		Digest: req.GetTag().GetDigest(),
	}

	storedTag, err := s.repository.UpsertRepositoryTag(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert tag %s: %w", tag.ID, err)
	}

	// Convert back to protobuf
	pbTag := &modelpb.RepositoryTag{
		Name:       storedTag.Name,
		Id:         storedTag.ID,
		Digest:     storedTag.Digest,
		UpdateTime: timestamppb.New(storedTag.UpdateTime),
	}

	return &modelpb.CreateRepositoryTagResponse{Tag: pbTag}, nil
}

// GetRepositoryTag retrieves the information of a repository tag.
func (s *service) GetRepositoryTag(ctx context.Context, req *modelpb.GetRepositoryTagRequest) (*modelpb.GetRepositoryTagResponse, error) {
	logger, _ := logx.GetZapLogger(ctx)

	name := utils.RepositoryTagName(req.GetName())
	repo, id, err := name.ExtractRepositoryAndID()
	if err != nil {
		return nil, fmt.Errorf("invalid tag name")
	}

	rt, err := s.repository.GetRepositoryTag(ctx, name)
	if err != nil {
		if !errors.Is(err, errorsx.ErrNotFound) {
			return nil, err
		}
		rt, err = s.populateMissingRepositoryTags(ctx, name, repo, id)
		if err != nil {
			logger.Warn(fmt.Sprintf("Create missing tag record error: %v", err))
			return nil, err
		}
	}

	pbTag := &modelpb.RepositoryTag{
		Name:       rt.Name,
		Id:         rt.ID,
		Digest:     rt.Digest,
		UpdateTime: timestamppb.New(rt.UpdateTime),
	}

	return &modelpb.GetRepositoryTagResponse{Tag: pbTag}, nil
}

// ListRepositoryTags fetches and paginates the tags of a repository in a
// remote distribution registry.
func (s *service) ListRepositoryTags(ctx context.Context, req *modelpb.ListRepositoryTagsRequest) (*modelpb.ListRepositoryTagsResponse, error) {
	logger, _ := logx.GetZapLogger(ctx)

	pageSize := pageSizeInRange(req.GetPageSize())
	page := pageInRange(req.GetPage())
	idx0, idx1 := page*pageSize, (page+1)*pageSize

	// Content registry repository, not to be mixed with s.repository (model
	// storage implementation).
	_, repo, ok := strings.Cut(req.GetParent(), "repositories/")
	if !ok {
		return nil, fmt.Errorf("namespace error")
	}

	registryClient := httpclient.NewRegistryClient(ctx, s.cfg.Registry.Host, s.cfg.Registry.Port)
	tagIDs, err := registryClient.ListTags(ctx, repo)
	if err != nil {
		return nil, err
	}

	totalSize := len(tagIDs)
	var paginatedIDs []string
	switch {
	case idx0 >= totalSize:
	case idx1 > totalSize:
		paginatedIDs = tagIDs[idx0:]
	default:
		paginatedIDs = tagIDs[idx0:idx1]
	}

	tags := make([]*modelpb.RepositoryTag, 0, len(paginatedIDs))
	for _, id := range paginatedIDs {
		name := utils.NewRepositoryTagName(repo, id)
		rt, err := s.repository.GetRepositoryTag(ctx, name)
		if err != nil {
			if !errors.Is(err, errorsx.ErrNotFound) {
				return nil, fmt.Errorf("failed to fetch tag %s: %w", id, err)
			}

			// The source of truth for tags is the registry. The local
			// repository only holds extra information we'll aggregate to the
			// tag ID list. If no record is found locally, we create the missing
			// record.
			rt, err = s.populateMissingRepositoryTags(ctx, name, repo, id)
			if err != nil {
				logger.Warn(fmt.Sprintf("Create missing tag record error: %v", err))
				rt = &datamodel.Tag{Name: string(name), ID: id}
			}
		}

		pbTag := &modelpb.RepositoryTag{
			Name:   rt.Name,
			Id:     rt.ID,
			Digest: rt.Digest,
		}
		if !rt.UpdateTime.IsZero() {
			pbTag.UpdateTime = timestamppb.New(rt.UpdateTime)
		}
		tags = append(tags, pbTag)
	}

	return &modelpb.ListRepositoryTagsResponse{
		PageSize:  int32(pageSize),
		Page:      int32(page),
		TotalSize: int32(totalSize),
		Tags:      tags,
	}, nil
}

func (s *service) populateMissingRepositoryTags(ctx context.Context, name utils.RepositoryTagName, repo string, id string) (*datamodel.Tag, error) {
	registryClient := httpclient.NewRegistryClient(ctx, s.cfg.Registry.Host, s.cfg.Registry.Port)
	digest, err := registryClient.GetTagDigest(ctx, repo, id)
	if err != nil {
		return nil, err
	}

	tag := &datamodel.Tag{
		Name:   string(name),
		ID:     id,
		Digest: digest,
	}

	if _, err := s.repository.UpsertRepositoryTag(ctx, tag); err != nil {
		return nil, err
	}

	// Fetch the tag again to get the update time
	rt, err := s.repository.GetRepositoryTag(ctx, name)
	if err != nil {
		return tag, nil // Return the tag without update time if fetch fails
	}

	return rt, nil
}

func pageSizeInRange(pageSize int32) int {
	const defaultPageSize = 10
	const maxPageSize = 100

	if pageSize <= 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return int(pageSize)
}

func pageInRange(page int32) int {
	if page < 0 {
		return 0
	}
	return int(page)
}
