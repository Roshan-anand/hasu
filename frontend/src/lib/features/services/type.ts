import type { AppServiceName, PsqlServiceName, ServiceType } from '@/types';
import type { DeploymentStatus } from '../deployments/type';

type ServiceListBase = {
	id: string;
	name: string;
	created_at: string;
};

type PredefinedServiceStatus = 'running' | 'paused';

export type ServiceListPSQL = ServiceListBase & {
	type: PsqlServiceName;
	status: PredefinedServiceStatus;
	volume: string;
};

export type ServiceListApp = ServiceListBase & {
	type: AppServiceName;
	gh_repo_name: string;
	gh_repo_url: string;
	git_provider: string;
	branch_name: string;
	replicas: number;
};

export type ServiceListResponse = ServiceListPSQL | ServiceListApp;

type DeleteServicePayload = {
	service_id: string;
};

export type DeleteAppServicePayload = DeleteServicePayload;
export type DeletePsqlServicePayload = DeleteServicePayload & {
	keep_data: boolean;
};

export type GitProviderKey = 'github' | 'gitlab' | 'bitbucket';

export type GitProviderOption = {
	// key: GitProviderKey;
	name: string;
	icon: string;
	listApi: string;
	createApi: string;
};

export type GithubApp = {
	name: string;
	app_id: number;
	created_at: string;
};

export type GithubRepo = {
	id: number;
	name: string;
	full_name: string;
	html_url: string;
	repo_url: string;
	private: boolean;
	default_branch: string;
	branches: string[];
};

export type GetReposPayload = { provider: GitProviderOption; appId: number };

export type CreateServiceResponse = {
	name: string;
	type: ServiceType;
};

export type CreateAppServiceBody = {
	name: string;
	git_provider: GitProviderKey;
	gh_app_id: number;
	gh_repo_id: number;
	default_branch: string;
	build_path: string;
	watch_path: string;
	public: boolean;
	port: number;
	docker_build: {
		file_path: string;
		context_path: string;
		build_stage: string;
	};
};

export type CreateAppServiceForm = CreateAppServiceBody & {
	env: string;
	build_args: string;
	build_secrets: string;
};

export type CreateServicePayload = CreateAppServiceBody & {
	env: string[];
	build_args: string[];
	build_secrets: string[];
	instance_id: string;
};

export type CreatePsqlServiceBody = {
	name: string;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
	// optional: name of an orphan volume to reattach instead of creating a new one
	volume?: string;
};

export type CreatePsqlServicePayload = CreatePsqlServiceBody & {
	instance_id: string;
};

export type PsqlServiceDetails = {
	id: string;
	name: string;
	swarm_service: string;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
	internal_url: string;
	volume: string;
	status: PredefinedServiceStatus;
	created_at: string;
};

export type UpdatePsqlServicePayload = {
	service_id: string;
	db_name: string;
	db_user: string;
	db_password: string;
};

export type RedeployPsqlServicePayload = {
	service_id: string;
};

export type AppServiceDetails = {
	id: string;
	name: string;
	gh_repo_name: string;
	gh_repo_url: string;
	status: DeploymentStatus;
	replicas: number;
	commit_msg: string;
	branch: string;
	swarm_service: string;
	is_public: boolean;
	domain: string;
	internal_url: string;
	port: number;
	created_at: string;
};

// service domain types
export type UpdateServiceDomainPayload = {
	service_id: string;
	domain: string;
	port: number;
	is_public: boolean;
};

// service env types
export type UpdateEnvPayload = {
	service_id: string;
	env: string[];
	build_args: string[];
	build_secrets: string[];
};

export type GetEnvRes = Omit<UpdateEnvPayload, 'service_id'>;

// navigation types
export type ServiceTab = '' | 'deployment' | 'env' | 'settings' | 'dependencies';

export type NavItem = {
	label: string;
	tab: ServiceTab;
};

export type AppServiceSettings = {
	domain: string;
	port: number;
	is_public: boolean;
	replicas: number;
};

export type ScaleAppServicePayload = {
	service_id: string;
	replicas: number;
};

export type DependencyTarget = {
	id: string;
	name: string;
	service_type: string;
	allowed_cols: string[];
};

export type ServiceDependency = {
	id: string;
	source_service_id: string;
	target_service_id: string;
	target_service_name: string;
	target_service_type: string;
	target_col: string;
	env_key: string;
	created_at: string;
	updated_at: string;
};

export type CreateDependencyPayload = {
	source_service_id: string;
	target_service_id: string;
	target_col: string;
	env_key: string;
};

export type UpdateDependencyPayload = {
	target_service_id: string;
	target_col: string;
	env_key: string;
};

export type PRInfo = {
	id: number;
	number: number;
	title: string;
	state: string;
	html_url: string;
	head_branch: string;
	repo_id: number;
};

// dependency graph types
export type GraphNode = {
	id: string;
	name: string;
	type: string;
	service_type: string;
};

export type GraphEdge = {
	source: string;
	target: string;
	target_col: string;
	env_key: string;
};

export type DependencyGraphResponse = {
	nodes: GraphNode[];
	edges: GraphEdge[];
};
