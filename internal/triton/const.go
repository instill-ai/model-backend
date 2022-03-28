package triton

type DetectionOutput struct {
	Boxes  [][][]float32
	Labels [][]string
}
