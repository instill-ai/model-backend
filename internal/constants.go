package utils

var CVTasks = map[string]int{
	"cls": 1,
	"det": 2,
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
