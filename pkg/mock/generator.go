package mock

//go:generate minimock -g -i github.com/instill-ai/model-backend/pkg/minio.MinioI -o ./ -s "_mock.gen.go"

// todo: port the `mockgen` generated files to `minimock`
//go:generate mockgen -destination mock_repository.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/repository Repository
