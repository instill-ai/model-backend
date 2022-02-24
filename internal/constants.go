package utils

var CVTasks = map[string]int{
	"CLASSIFICATION": 1,
	"DETECTION":      2,
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
