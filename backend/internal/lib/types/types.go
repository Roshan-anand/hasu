package types

type AppEnv string

const (
	DevMode  AppEnv = "development"
	ProdMode AppEnv = "production"
	TestMode AppEnv = "test"
)

type UserRole string

const (
	AdminRole  UserRole = "admin"
	MemberRole UserRole = "member"
)

type ServiceType string

const (
	PsqlServiceType  ServiceType = "psql"
	AppServiceType   ServiceType = "app"
	RedisServiceType ServiceType = "redis"
)

type PredefServiceType string

const (
	PSQLPredefServiceType  PredefServiceType = "psql"
	RedisPredefServiceType PredefServiceType = "redis"
	MongoPredefServiceType PredefServiceType = "mongodb"
)

type DeploymentStatus string

const (
	DeploymentBuilding DeploymentStatus = "building"
	DeploymentReady    DeploymentStatus = "ready"
	DeploymentError    DeploymentStatus = "error"
	DeploymentQueued   DeploymentStatus = "queued"
	DeploymentInactive DeploymentStatus = "inactive" // service is not using this image (can be rollbacked)
	DeploymentPruned   DeploymentStatus = "pruned"   // image is not available
	DeploymentPaused   DeploymentStatus = "paused"
)

type PredefinedServiceStatus string

const (
	PredefServiceRunning PredefinedServiceStatus = "running"
	PredefServicePaused  PredefinedServiceStatus = "paused"
)

type Res[T any] struct {
	Message string `json:"message" validate:"required"`
	Data    T      `json:"data"`
}
