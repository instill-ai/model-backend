package util

var Tasks = map[string]int{
	"TASK_CLASSIFICATION": 1,
	"TASK_DETECTION":      2,
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
