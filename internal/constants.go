package utils

var Tasks = map[string]int{
	"TASK_CLASSIFICATION": 1,
	"TASK_DETECTION":      2,
}

const MaxBatchSize int = 32

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const MaxImageSizeBytes int = 4 * MB
