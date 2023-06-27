package handler

import mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"

func GenOwnerPermalink(owner *mgmtPB.User) string {
	return "users/" + owner.GetUid()
}
