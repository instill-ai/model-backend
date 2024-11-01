package mock

//go:generate minimock -g -i github.com/instill-ai/model-backend/pkg/repository.Repository -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/model-backend/pkg/ray.Ray -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/model-backend/pkg/acl.ACLClientInterface -o ./ -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/protogen-go/artifact/artifact/v1alpha.ArtifactPrivateServiceClient -o ./ -s "_mock.gen.go"
