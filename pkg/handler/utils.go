package handler

import (
	"context"

	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/service"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func parseView(view modelpb.View) modelpb.View {
	if view == modelpb.View_VIEW_UNSPECIFIED {
		return modelpb.View_VIEW_BASIC
	}
	return view
}

func authenticateUser(ctx context.Context, allowVisitor bool) error {
	if resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey) == "user" {
		if resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey) == "" {
			return service.ErrUnauthenticated
		}
		return nil
	} else {
		if !allowVisitor {
			return service.ErrUnauthenticated
		}
		if resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey) == "" {
			return service.ErrUnauthenticated
		}
		return nil
	}
}
