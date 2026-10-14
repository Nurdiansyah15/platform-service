package dto

const (
	HealthStatusOK       = "ok"
	HealthStatusDegraded = "degraded"
)

type HealthResponse struct {
	Status         string `json:"status"`
	Database       string `json:"database"`
	DeploymentMode string `json:"deployment_mode"`
}
