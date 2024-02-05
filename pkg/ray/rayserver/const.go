package rayserver

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
	Status            string                           `json:"status,omitempty"`
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
	Name                 string               `json:"name,omitempty"`
	Status               string               `json:"status,omitempty"`
	Message              string               `json:"message,omitempty"`
	HTTPDeploymentConfig HTTPDeploymentConfig `json:"deployment_config,omitempty"`
	Replicas             []Replica            `json:"replicas,omitempty"`
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
	NodeID      string  `json:"node_id,omitempty"`
	NodeIP      string  `json:"node_ip,omitempty"`
	ActorID     string  `json:"actor_id,omitempty"`
	ActorName   string  `json:"actor_name,omitempty"`
	WorkerID    string  `json:"worker_id,omitempty"`
	LogFilePath string  `json:"log_file_path,omitempty"`
	ReplicaID   string  `json:"replica_id,omitempty"`
	State       string  `json:"state,omitempty"`
	PID         float64 `json:"pid,omitempty"`
	StartTimeS  float64 `json:"start_time_s,omitempty"`
}
