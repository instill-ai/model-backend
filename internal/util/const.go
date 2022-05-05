package util

import modelPB "github.com/instill-ai/protogen-go/model/v1alpha"

var Tasks = map[string]modelPB.ModelInstance_Task{
	"TASK_CLASSIFICATION": modelPB.ModelInstance_TASK_CLASSIFICATION,
	"TASK_DETECTION":      modelPB.ModelInstance_TASK_DETECTION,
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

const USER_ID = "2a06c2f7-8da9-4046-91ea-240f88a5d729"
const TYPE_USER = "user"
