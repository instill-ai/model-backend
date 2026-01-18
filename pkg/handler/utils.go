package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/instill-ai/x/constant"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	errorsx "github.com/instill-ai/x/errors"
	resourcex "github.com/instill-ai/x/resource"
)

func parseView(view modelpb.View) modelpb.View {
	if view == modelpb.View_VIEW_UNSPECIFIED {
		return modelpb.View_VIEW_BASIC
	}
	return view
}

// parseNamespaceFromParent extracts namespace ID from parent string.
// Format: namespaces/{namespace}
func parseNamespaceFromParent(parent string) (string, error) {
	parts := strings.Split(parent, "/")
	if len(parts) < 2 || parts[0] != "namespaces" {
		return "", fmt.Errorf("invalid parent format: %s", parent)
	}
	return parts[1], nil
}

// parseModelFromName extracts namespace ID and model ID from name string.
// Format: namespaces/{namespace}/models/{model}
func parseModelFromName(name string) (namespaceID, modelID string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) < 4 || parts[0] != "namespaces" || parts[2] != "models" {
		return "", "", fmt.Errorf("invalid model name format: %s", name)
	}
	return parts[1], parts[3], nil
}

// parseModelVersionFromName extracts namespace ID, model ID, and version from name string.
// Format: namespaces/{namespace}/models/{model}/versions/{version}
func parseModelVersionFromName(name string) (namespaceID, modelID, version string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) < 6 || parts[0] != "namespaces" || parts[2] != "models" || parts[4] != "versions" {
		return "", "", "", fmt.Errorf("invalid model version name format: %s", name)
	}
	return parts[1], parts[3], parts[5], nil
}

// parseModelFromParent extracts namespace ID and model ID from parent string for model versions.
// Format: namespaces/{namespace}/models/{model}
func parseModelFromParent(parent string) (namespaceID, modelID string, err error) {
	parts := strings.Split(parent, "/")
	if len(parts) < 4 || parts[0] != "namespaces" || parts[2] != "models" {
		return "", "", fmt.Errorf("invalid parent format: %s", parent)
	}
	return parts[1], parts[3], nil
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
