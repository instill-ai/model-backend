package mock

//go:generate mockgen -destination service_mock.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/service Service
//go:generate mockgen -destination ray_mock.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/ray Ray
//go:generate mockgen -destination repository_mock.go -package $GOPACKAGE github.com/instill-ai/model-backend/pkg/repository Repository
//go:generate mockgen -destination fga_client_mock.go -package $GOPACKAGE github.com/openfga/api/proto/openfga/v1 OpenFGAServiceClient
//go:generate mockgen -destination mgmt_private_service_client_mock.go -package $GOPACKAGE github.com/instill-ai/protogen-go/core/mgmt/v1beta MgmtPrivateServiceClient
