package util

import modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"

var Tasks = map[string]modelPB.ModelInstance_Task{
	"TASK_CLASSIFICATION": modelPB.ModelInstance_TASK_CLASSIFICATION,
	"TASK_DETECTION":      modelPB.ModelInstance_TASK_DETECTION,
	"TASK_KEYPOINT":       modelPB.ModelInstance_TASK_KEYPOINT,
}

var Visibility = map[string]modelPB.Model_Visibility{
	"public":  modelPB.Model_VISIBILITY_PUBLIC,
	"private": modelPB.Model_VISIBILITY_PRIVATE,
}

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const MaxBatchSize int = 32
const MaxImageSizeBytes int = 4 * MB

const DEFAULT_GCP_SERVICE_ACCOUNT_FILE = "https://artifacts.instill.tech/default-service-account.json"
