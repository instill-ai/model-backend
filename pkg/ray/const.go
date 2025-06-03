package ray

type GetApplicationStatus struct {
	ControllerInfo ControllerInfo         `json:"controller_info,omitempty"`
	ProxyLocation  string                 `json:"proxy_location,omitempty"`
	HTTPOptions    HTTPOptions            `json:"http_options,omitempty"`
	GrpcOptions    GrpcOptions            `json:"grpc_options,omitempty"`
	Proxies        map[string]Proxy       `json:"proxies,omitempty"`
	DeployMode     string                 `json:"deploy_mode,omitempty"`
	Applications   map[string]Application `json:"applications,omitempty"`
}

type ControllerInfo struct {
	NodeID      string `json:"node_id,omitempty"`
	NodeIP      string `json:"node_ip,omitempty"`
	ActorID     string `json:"actor_id,omitempty"`
	ActorName   string `json:"actor_name,omitempty"`
	WorkerID    string `json:"worker_id,omitempty"`
	LogFilePath string `json:"log_file_path,omitempty"`
}

type HTTPOptions struct {
	Host              string  `json:"host,omitempty"`
	Port              float64 `json:"port,omitempty"`
	RootPath          string  `json:"root_path,omitempty"`
	RequestTimeoutS   float64 `json:"request_timeout_s,omitempty"`
	KeepAliveTimeoutS float64 `json:"keep_alive_timeout_s,omitempty"`
}

type GrpcOptions struct {
	Port                  float64  `json:"port,omitempty"`
	GrpcServicerFunctions []string `json:"grpc_servicer_functions,omitempty"`
}

type Proxy struct {
	NodeID      string `json:"node_id,omitempty"`
	NodeIP      string `json:"node_ip,omitempty"`
	ActorID     string `json:"actor_id,omitempty"`
	ActorName   string `json:"actor_name,omitempty"`
	WorkerID    string `json:"worker_id,omitempty"`
	LogFilePath string `json:"log_file_path,omitempty"`
	Status      string `json:"status,omitempty"`
}

type Application struct {
	Name              string                           `json:"name,omitempty"`
	RoutePrefix       string                           `json:"route_prefix,omitempty"`
	DocsPath          string                           `json:"docs_path,omitempty"`
	Status            ApplicationStatusStr             `json:"status,omitempty"`
	Message           string                           `json:"message,omitempty"`
	LastDeployedTimeS float64                          `json:"last_deployed_time_s,omitempty"`
	DeployedAppConfig DeployedAppConfig                `json:"deployed_app_config,omitempty"`
	Deployments       map[string]ApplicationDeployment `json:"deployments,omitempty"`
}

type DeployedAppConfig struct {
	Name        string                  `json:"name,omitempty"`
	RoutePrefix string                  `json:"route_prefix,omitempty"`
	ImportPath  string                  `json:"import_path,omitempty"`
	Deployments []DeployedAppDeployment `json:"deployments,omitempty"`
}
type DeployedAppDeployment struct {
	Name        string         `json:"name,omitempty"`
	NumReplicas string         `json:"num_replicas,omitempty"`
	UserConfig  map[string]any `json:"user_config,omitempty"`
}
type ApplicationDeployment struct {
	Name                 string                     `json:"name,omitempty"`
	Status               DeploymentStatusStr        `json:"status,omitempty"`
	StatusTrigger        DeploymentStatusTriggerStr `json:"status_trigger,omitempty"`
	Message              string                     `json:"message,omitempty"`
	HTTPDeploymentConfig HTTPDeploymentConfig       `json:"deployment_config,omitempty"`
	TargetNumReplicas    int                        `json:"target_num_replicas,omitempty"`
	Replicas             []Replica                  `json:"replicas,omitempty"`
}

type HTTPDeploymentConfig struct {
	Name                      string         `json:"name,omitempty"`
	NumReplicas               float64        `json:"num_replicas,omitempty"`
	MaxConcurrentQueries      float64        `json:"max_concurrent_queries,omitempty"`
	UserConfig                map[string]any `json:"user_config,omitempty"`
	GracefulShutdownWaitLoopS float64        `json:"graceful_shutdown_wait_loop_s,omitempty"`
	GracefulShutdownTimeoutS  float64        `json:"graceful_shutdown_timeout_s,omitempty"`
	HealthCheckPeriodS        float64        `json:"health_check_period_s,omitempty"`
	HealthCheckTimeoutS       float64        `json:"health_check_timeout_s,omitempty"`
	RayActorOptions           map[string]any `json:"ray_actor_options,omitempty"`
	IsDriverDeployment        bool           `json:"is_driver_deployment,omitempty"`
}

type Replica struct {
	NodeID      string          `json:"node_id,omitempty"`
	NodeIP      string          `json:"node_ip,omitempty"`
	ActorID     string          `json:"actor_id,omitempty"`
	ActorName   string          `json:"actor_name,omitempty"`
	WorkerID    string          `json:"worker_id,omitempty"`
	LogFilePath string          `json:"log_file_path,omitempty"`
	ReplicaID   string          `json:"replica_id,omitempty"`
	State       ReplicaStateStr `json:"state,omitempty"`
	PID         float64         `json:"pid,omitempty"`
	StartTimeS  float64         `json:"start_time_s,omitempty"`
}

type ApplicationStatusStr string

const (
	ApplicationStatusStrNotStarted   ApplicationStatusStr = "NOT_STARTED"
	ApplicationStatusStrDeploying    ApplicationStatusStr = "DEPLOYING"
	ApplicationStatusStrDeployFailed ApplicationStatusStr = "DEPLOY_FAILED"
	ApplicationStatusStrRunning      ApplicationStatusStr = "RUNNING"
	ApplicationStatusStrUnhealthy    ApplicationStatusStr = "UNHEALTHY"
	ApplicationStatusStrDeleting     ApplicationStatusStr = "DELETING"
)

type DeploymentStatusStr string

const (
	DeploymentStatusStrUpdating    DeploymentStatusStr = "UPDATING"
	DeploymentStatusStrHealthy     DeploymentStatusStr = "HEALTHY"
	DeploymentStatusStrUnhealthy   DeploymentStatusStr = "UNHEALTHY"
	DeploymentStatusStrUpscaling   DeploymentStatusStr = "UPSCALING"
	DeploymentStatusStrDownscaling DeploymentStatusStr = "DOWNSCALING"
)

type DeploymentStatusTriggerStr string

const (
	DeploymentStatusTriggerStrUnspecified           DeploymentStatusTriggerStr = "UNSPECIFIED"
	DeploymentStatusTriggerStrConfigUpdateStarted   DeploymentStatusTriggerStr = "CONFIG_UPDATE_STARTED"
	DeploymentStatusTriggerStrConfigUpdateCompleted DeploymentStatusTriggerStr = "CONFIG_UPDATE_COMPLETED"
	DeploymentStatusTriggerStrUpscaleCompleted      DeploymentStatusTriggerStr = "UPSCALE_COMPLETED"
	DeploymentStatusTriggerStrDownscaleCompleted    DeploymentStatusTriggerStr = "DOWNSCALE_COMPLETED"
	DeploymentStatusTriggerStrAutoscaling           DeploymentStatusTriggerStr = "AUTOSCALING"
	DeploymentStatusTriggerStrReplicaStartupFailed  DeploymentStatusTriggerStr = "REPLICA_STARTUP_FAILED"
	DeploymentStatusTriggerStrHealthCheckFailed     DeploymentStatusTriggerStr = "HEALTH_CHECK_FAILED"
	DeploymentStatusTriggerStrInternalError         DeploymentStatusTriggerStr = "INTERNAL_ERROR"
	DeploymentStatusTriggerStrDeleting              DeploymentStatusTriggerStr = "DELETING"
)

type ReplicaStateStr string

const (
	ReplicaStateStrStarting         ReplicaStateStr = "STARTING"
	ReplicaStateStrUpdating         ReplicaStateStr = "UPDATING"
	ReplicaStateStrRecovering       ReplicaStateStr = "RECOVERING"
	ReplicaStateStrRunning          ReplicaStateStr = "RUNNING"
	ReplicaStateStrStopping         ReplicaStateStr = "STOPPING"
	ReplicaStateStrPendingMigration ReplicaStateStr = "PENDING_MIGRATION"
)

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
		PromptImage: "https://artifacts.instill-ai.com/imgs/dog.jpg",
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
		PromptImages:  "https://artifacts.instill-ai.com/imgs/dog.jpg",
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
		ImgURL: "https://artifacts.instill-ai.com/imgs/dog.jpg",
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
	RayApplications []RayApplication `yaml:"ray_applications" json:"applications"`
}

type Action string

const (
	Sync     Action = "sync"
	Deploy   Action = "deploy"
	Undeploy Action = "undeploy"
	UpScale  Action = "upscale"
)

type ApplicationWithAction struct {
	RayApplication RayApplication
	Action         Action
}

type RayApplication struct {
	Name        string     `yaml:"name" json:"name"`
	ImportPath  string     `yaml:"import_path" json:"import_path"`
	RoutePrefix string     `yaml:"route_prefix" json:"route_prefix"`
	RuntimeEnv  RuntimeEnv `yaml:"runtime_env" json:"runtime_env"`
}

type RuntimeEnv struct {
	ImageURI string            `yaml:"image_uri" json:"image_uri"`
	EnvVars  map[string]string `yaml:"env_vars" json:"env_vars"`
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
	// Ray redis key
	RayDeploymentKey = "model_deployment_config"

	// Ray deployment env variables
	EnvIsTestModel        = "RAY_IS_TEST_MODEL"
	EnvIsHighScaleModel   = "RAY_IS_HIGH_SCALE_MODEL"
	EnvMemory             = "RAY_MEMORY"
	EnvTotalVRAM          = "RAY_TOTAL_VRAM"
	EnvRayAcceleratorType = "RAY_ACCELERATOR_TYPE"
	EnvRayCustomResource  = "RAY_CUSTOM_RESOURCE"
	EnvNumOfGPUs          = "RAY_NUM_OF_GPUS"
	EnvNumOfCPUs          = "RAY_NUM_OF_CPUS"
	EnvNumOfMinReplicas   = "RAY_NUM_OF_MIN_REPLICAS"
	EnvNumOfMaxReplicas   = "RAY_NUM_OF_MAX_REPLICAS"
	DummyModelPrefix      = "dummy-"
)
