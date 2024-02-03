package handler

import (
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func parseView(view modelPB.View) modelPB.View {
	if view == modelPB.View_VIEW_UNSPECIFIED {
		return modelPB.View_VIEW_BASIC
	}
	return view
}
