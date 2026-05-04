import type { ServiceType } from '@/types';

export type CreateServiceResponse = {
	id: string;
	type: ServiceType;
};

export type ServiceRow = {
	id: string;
	type: ServiceType;
	name: string;
	description: string;
	created_at: string;
};

export type ServiceListResponse = {
	services: ServiceRow[];
};

export type DeleteServicePayload = {
	service_id: string;
	type: ServiceType;
};

export type GitProviderKey = 'github' | 'gitlab' | 'bitbucket';

export type GitProviderOption = {
	key: GitProviderKey;
	name: string;
	icon: string;
	api: string;
};

export type ApiMessageRes = {
	message: string;
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
};

export type GetRepoResult = {
	status: number;
	repos: GithubRepo[];
	message: string;
	provider: GitProviderKey;
};

export type CreateAppServiceBody = {
	org_id: string;
	name: string;
	git_provider: GitProviderKey;
	gh_app_id: number;
	git_repo_id: string;
	git_repo_name: string;
	git_repo_url: string;
	default_branch: string;
	build_path: string;
	watch_path: string;
};

export type UpdateAppServiceBody = {
	service_id: string;
	git_provider: GitProviderKey;
	gh_app_id: number;
	git_repo_id: string;
	git_repo_name: string;
	git_repo_url: string;
	default_branch: string;
	build_path: string;
	watch_path: string;
};

export type CreatePsqlServiceBody = {
	org_id: string;
	name: string;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
};

export type CreateServicePayload =
	| { type: 'app'; body: CreateAppServiceBody }
	| { type: 'psql'; body: CreatePsqlServiceBody };
