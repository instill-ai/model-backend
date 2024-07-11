package ray

type InferInput any

type TextToImageInput struct {
	Prompt      string
	PromptImage string
	Steps       int32
	CfgScale    float32
	Seed        int32
	Samples     int32
	ExtraParams string
}

func NewTextToImageInput() *TextToImageInput {
	return &TextToImageInput{
		Prompt:   "A cute cat, sleeping.",
		Steps:    10,
		CfgScale: 7,
		Seed:     1024,
		Samples:  1,
	}
}

type ImageToImageInput struct {
	Prompt      string
	PromptImage string
	Steps       int32
	CfgScale    float32
	Seed        int32
	Samples     int32
	ExtraParams string
}

func NewImageToImageInput() *ImageToImageInput {
	return &ImageToImageInput{
		Prompt:      "cute dog",
		PromptImage: "https://artifacts.instill.tech/imgs/dog.jpg",
		Steps:       10,
		CfgScale:    7,
		Seed:        1024,
		Samples:     1,
	}
}

type TextGenerationInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

func NewTextGenerationInput() *TextGenerationInput {
	return &TextGenerationInput{
		Prompt:        "Simply put, the theory of relativity states that",
		SystemMessage: "You are a helpful assistant.",
		MaxNewTokens:  50,
		TopK:          10,
		Temperature:   0.7,
		Seed:          1024,
	}
}

type TextGenerationChatInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

func NewTextGenerationChatInput() *TextGenerationChatInput {
	return &TextGenerationChatInput{
		Prompt:        "Who are you?",
		SystemMessage: "You are a lovely cat, named Penguin.",
		MaxNewTokens:  512,
		TopK:          10,
		Temperature:   0.7,
		Seed:          1024,
	}
}

type VisualQuestionAnsweringInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

func NewVisualQuestionAnsweringInput() *VisualQuestionAnsweringInput {
	return &VisualQuestionAnsweringInput{
		Prompt:        "What is in the picture?",
		PromptImages:  "https://artifacts.instill.tech/imgs/dog.jpg",
		SystemMessage: "You are a helpful assistant.",
		MaxNewTokens:  512,
		Temperature:   0.7,
		TopK:          10,
		Seed:          1024,
	}
}

type ImageInput struct {
	ImgURL    string
	ImgBase64 string
}

func NewImageInput() *ImageInput {
	return &ImageInput{
		ImgURL: "https://artifacts.instill.tech/imgs/dog.jpg",
	}
}

type DetectionOutput struct {
	Boxes  [][][]float32
	Labels [][]string
}

type OcrOutput struct {
	Boxes  [][][]float32
	Texts  [][]string
	Scores [][]float32
}

type KeypointOutput struct {
	Keypoints [][][][]float32
	Boxes     [][][]float32
	Scores    [][]float32
}

type BatchUnspecifiedTaskOutputs struct {
	Name              string
	Shape             []int64
	DataType          string
	SerializedOutputs []any // batching output
}

type SingleOutputUnspecifiedTaskOutput struct {
	Name     string
	Shape    []int64
	DataType string
	Data     any // batching output
}

type UnspecifiedTaskOutput struct {
	RawOutput []SingleOutputUnspecifiedTaskOutput
}

type InstanceSegmentationOutput struct {
	Rles   [][]string
	Boxes  [][][]float32
	Scores [][]float32
	Labels [][]string
}

type SemanticSegmentationOutput struct {
	Rles       [][]string
	Categories [][]string
}

type TextToImageOutput struct {
	Images [][]string
}

type ImageToImageOutput struct {
	Images [][]string
}

type TextGenerationOutput struct {
	Text []string
}

type TextGenerationChatOutput struct {
	Text []string
}

type VisualQuestionAnsweringOutput struct {
	Text []string
}

type ModelDeploymentConfig struct {
	Applications []Application `yaml:"applications" json:"applications"`
}

type Action string

const (
	Deploy   Action = "deploy"
	Undeploy Action = "undeploy"
	Sync     Action = "sync"
)

type ApplicationWithAction struct {
	Application Application
	Action      Action
}

type Application struct {
	Name        string     `yaml:"name" json:"name"`
	ImportPath  string     `yaml:"import_path" json:"import_path"`
	RoutePrefix string     `yaml:"route_prefix" json:"route_prefix"`
	RuntimeEnv  RuntimeEnv `yaml:"runtime_env" json:"runtime_env"`
}

type RuntimeEnv struct {
	Container Container `yaml:"container" json:"container"`
}

type Container struct {
	Image      string   `yaml:"image" json:"image"`
	RunOptions []string `yaml:"run_options" json:"run_options"`
}

var SupportedAcceleratorType = map[string]string{
	"CPU":                       "CPU",
	"GPU":                       "GPU",
	"NVIDIA_TESLA_V100":         "V100",
	"NVIDIA_TESLA_P100":         "P100",
	"NVIDIA_TESLA_T4":           "T4",
	"NVIDIA_TESLA_P4":           "P4",
	"NVIDIA_TESLA_K80":          "K80",
	"NVIDIA_TESLA_A10G":         "A10G",
	"NVIDIA_L4":                 "L4",
	"NVIDIA_A100":               "A100",
	"INTEL_MAX_1550":            "Intel-GPU-Max-1550",
	"INTEL_MAX_1100":            "Intel-GPU-Max-1100",
	"INTEL_GAUDI":               "Intel-GAUDI",
	"AMD_INSTINCT_MI100":        "AMD-Instinct-MI100",
	"AMD_INSTINCT_MI250X":       "AMD-Instinct-MI250X",
	"AMD_INSTINCT_MI250":        "AMD-Instinct-MI250X-MI250",
	"AMD_INSTINCT_MI210":        "AMD-Instinct-MI210",
	"AMD_INSTINCT_MI300X":       "AMD-Instinct-MI300X-OAM",
	"AMD_RADEON_R9_200_HD_7900": "AMD-Radeon-R9-200-HD-7900",
	"AMD_RADEON_HD_7900":        "AMD-Radeon-HD-7900",
	"AWS_NEURON_CORE":           "aws-neuron-core",
	"GOOGLE_TPU_V2":             "TPU-V2",
	"GOOGLE_TPU_V3":             "TPU-V3",
	"GOOGLE_TPU_V4":             "TPU-V4",
	"NVIDIA_A100_40G":           "A100-40G",
	"NVIDIA_A100_80G":           "A100-80G",
}

var SupportedAcceleratorTypeMemory = map[string]int{
	"NVIDIA_TESLA_V100":         16,
	"NVIDIA_TESLA_P100":         16,
	"NVIDIA_TESLA_T4":           16,
	"NVIDIA_TESLA_P4":           8,
	"NVIDIA_TESLA_K80":          24,
	"NVIDIA_TESLA_A10G":         24,
	"NVIDIA_L4":                 24,
	"NVIDIA_A100":               40,
	"INTEL_MAX_1550":            128,
	"INTEL_MAX_1100":            48,
	"INTEL_GAUDI":               128,
	"AMD_INSTINCT_MI100":        32,
	"AMD_INSTINCT_MI250X":       128,
	"AMD_INSTINCT_MI250":        128,
	"AMD_INSTINCT_MI210":        64,
	"AMD_INSTINCT_MI300X":       192,
	"AMD_RADEON_R9_200_HD_7900": 3,
	"AMD_RADEON_HD_7900":        3,
	"AWS_NEURON_CORE":           32,
	"GOOGLE_TPU_V2":             8,
	"GOOGLE_TPU_V3":             16,
	"GOOGLE_TPU_V4":             32,
	"NVIDIA_A100_40G":           40,
	"NVIDIA_A100_80G":           80,
}

const (
	EnvIsTestModel        = "RAY_IS_TEST_MODEL"
	EnvMemory             = "RAY_MEMORY"
	EnvTotalVRAM          = "RAY_TOTAL_VRAM"
	EnvRayAcceleratorType = "RAY_ACCELERATOR_TYPE"
	EnvRayCustomResource  = "RAY_CUSTOM_RESOURCE"
	EnvNumOfGPUs          = "RAY_NUM_OF_GPUS"
	EnvNumOfCPUs          = "RAY_NUM_OF_CPUS"
	EnvNumOfMinReplicas   = "RAY_NUM_OF_MIN_REPLICAS"
	EnvNumOfMaxReplicas   = "RAY_NUM_OF_MAX_REPLICAS"
)
