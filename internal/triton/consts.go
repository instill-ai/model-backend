package triton

const maxBatchSize int = 32

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const maxImageSizeBytes int = 4 * MB

const classificationTopK int64 = 5

type DetectionOutput struct {
	Boxes  [][][]float32
	Labels [][]string
}
