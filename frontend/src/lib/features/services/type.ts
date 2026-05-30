import type { AppServiceName, PsqlServiceName, ServiceType } from '@/types';
import type { DeploymentStatus } from '../deployments/type';

export type CreateServiceResponse = {
	id: string;
	type: ServiceType;
};

type ServiceListBase = {
	id: string;
	name: string;
	created_at: string;
};

export type ServiceListResponse =
	| (ServiceListBase & {
			type: PsqlServiceName;
	  })
	| (ServiceListBase & {
			type: AppServiceName;
			gh_repo_name: string;
			gh_repo_url: string;
			git_provider: string;
			branch_name: string;
	  });

export type DeleteServicePayload = {
	service_id: string;
	type: ServiceType;
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

export type CreateAppServiceBody = {
	name: string;
	git_provider: GitProviderKey;
	gh_app_id: number;
	gh_repo_id: number;
	default_branch: string;
	build_path: string;
	watch_path: string;
	public: boolean;
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
	org_id: string;
};

export type CreateServicePayload = CreateAppServiceBody & {
	env: string[];
	build_args: string[];
	build_secrets: string[];
	project_id: string;
};

export type CreatePsqlServiceBody = {
	name: string;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
};

export type CreatePsqlServicePayload = CreatePsqlServiceBody & {
	project_id: string;
};

export type PsqlServiceDetails = {
	id: string;
	name: string;
	swarm_service_name: string;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
	internal_url: string;
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
	commit_msg: string;
	branch_id: string;
	branch_name: string;
	domain: string;
	created_at: string;
};

// branch domain types
export type BranchDomainDetails = {
	branch_id: string;
	branch_name: string;
	domain: string;
	port: number;
};

export type UpdateBranchDomainPayload = Omit<BranchDomainDetails, 'branch_name'>;

export type GetBranchDomainRes = BranchDomainDetails[];

// service env types
export type UpdateEnvPayload = {
	service_id: string;
	env: string[];
	build_args: string[];
	build_secrets: string[];
};

export type GetEnvRes = Omit<UpdateEnvPayload, 'service_id'>;

// navigation types
export type ServiceTab = '' | 'deployment' | 'env' | 'domains';

export type NavItem = {
	label: string;
	tab: ServiceTab;
};
