package types

type AppEnv string

const (
	DevMode  AppEnv = "development"
	ProdMode AppEnv = "production"
)

type UserRole string

const (
	AdminRole  UserRole = "admin"
	MemberRole UserRole = "member"
)

type ServiceType string

const (
	PsqlServiceType ServiceType = "psql"
	AppServiceType  ServiceType = "app"
)

type DeploymentStatus string

const (
	DeploymentBuilding DeploymentStatus = "building"
	DeploymentReady    DeploymentStatus = "ready"
	DeploymentError    DeploymentStatus = "error"
	DeploymentQueued   DeploymentStatus = "queued"
	DeploymentInactive DeploymentStatus = "inactive" // service is not using this image (can be rollbacked)
	DeploymentPruned   DeploymentStatus = "pruned"   // image is not available
)
