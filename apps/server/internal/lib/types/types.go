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

type GitProvider string

const (
	GitHubProvider   GitProvider = "github"
	GitLabProvider   GitProvider = "gitlab"
	GitLocalProvider GitProvider = "local" // ! used only for local testing, not for production
)

type InstanceStatus string

const (
	InstanceCreating InstanceStatus = "creating"
	InstanceReady    InstanceStatus = "ready"
	InstanceDeleting InstanceStatus = "deleting"
)

type GitSourceType string

const (
	GitSourcePR     GitSourceType = "pr"
	GitSourceBranch GitSourceType = "branch"
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
	DeploymentCanceled DeploymentStatus = "canceled"
)

type PredefinedServiceStatus string

const (
	PredefServiceRunning PredefinedServiceStatus = "running"
	PredefServicePaused  PredefinedServiceStatus = "paused"
)

type CreatedBy string

const (
	CreatedByManual  CreatedBy = "manual"
	CreatedByWebhook CreatedBy = "webhook"
)

type PRState string

const (
	PROpen   PRState = "open"
	PRClosed PRState = "closed"
)

type DependencyTargetCol string

const (
	TargetColInternalURL DependencyTargetCol = "internal_url"
	TargetColDomain      DependencyTargetCol = "domain"
	TargetColDbName      DependencyTargetCol = "db_name"
	TargetColDbUser      DependencyTargetCol = "db_user"
	TargetColDbPassword  DependencyTargetCol = "db_password"
	TargetColPassword    DependencyTargetCol = "password"
	TargetColName        DependencyTargetCol = "name"
)

// Ptr returns a pointer to v. Useful for constructing nullable sqlc overrides.
func Ptr[T ~string](v T) *T { return &v }

type Res[T any] struct {
	Message string `json:"message" validate:"required"`
	Data    T      `json:"data"`
}

const ()
