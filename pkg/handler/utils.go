package handler

import (
	"context"

	"github.com/instill-ai/x/constant"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	errorsx "github.com/instill-ai/x/errors"
	resourcex "github.com/instill-ai/x/resource"
)

func parseView(view modelpb.View) modelpb.View {
	if view == modelpb.View_VIEW_UNSPECIFIED {
		return modelpb.View_VIEW_BASIC
	}
	return view
}

func authenticateUser(ctx context.Context, allowVisitor bool) error {
	if resourcex.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey) == "user" {
		if resourcex.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey) == "" {
			return errorsx.ErrUnauthenticated
		}
		return nil
	} else {
		if !allowVisitor {
			return errorsx.ErrUnauthenticated
		}
		if resourcex.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey) == "" {
			return errorsx.ErrUnauthenticated
		}
		return nil
	}
}
