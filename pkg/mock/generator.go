package mock

//go:generate mockgen -destination service_mock.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/service Service
//go:generate mockgen -destination ray_mock.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/ray Ray
