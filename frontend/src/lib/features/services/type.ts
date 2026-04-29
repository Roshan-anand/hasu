import type { ServiceType } from '@/types';

export interface CreateServiceResponse {
	id: string;
	type: ServiceType;
}

export type GitProviderKey = 'github' | 'gitlab' | 'bitbucket';

export interface GitProviderOption {
	key: GitProviderKey;
	name: string;
	icon: string;
	api: string;
}

export interface ApiMessageRes {
	message: string;
}

export interface GithubApp {
	name: string;
	app_id: number;
	created_at: string;
}

export interface GithubRepo {
	id: number;
	name: string;
	full_name: string;
	html_url: string;
	repo_url: string;
	private: boolean;
	default_branch: string;
}

export interface GetRepoResult {
	status: number;
	repos: GithubRepo[];
	message: string;
	provider: GitProviderKey;
}

export interface CreateAppServiceBody {
	project_id: string;
	name: string;
	description: string;
	app_name: string;
	git_provider: GitProviderKey;
	gh_app_id: number;
	git_repo_id: string;
	git_repo_name: string;
	git_repo_url: string;
	git_branch: string;
	build_path: string;
}

export interface CreatePsqlServiceBody {
	project_id: string;
	name: string;
	description: string;
	app_name: string;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
}

export type CreateServicePayload =
	| { type: 'app'; body: CreateAppServiceBody }
	| { type: 'psql'; body: CreatePsqlServiceBody };
