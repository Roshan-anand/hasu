import type { AppServiceName, PsqlServiceName, ServiceType } from '@/types';

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
	name: string;
	git_provider: GitProviderKey;
	gh_app_id: string;
	git_repo_id: string;
	git_repo_name: string;
	git_repo_url: string;
	build_path: string;
	watch_path: string;
	env: string;
	build_args: string;
	build_secrets: string;
};

export type CreatePsqlServiceBody = {
	name: string;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
};

export type CreateServicePayload =
	| {
			type: 'app';
			body: CreateAppServiceBody & {
				org_id: string;
				default_branch: string;
			};
	  }
	| {
			type: 'psql';
			body: CreatePsqlServiceBody & {
				org_id: string;
			};
	  };

export type CreateServiceForm =
	| (CreateAppServiceBody & { type: 'app' })
	| (CreatePsqlServiceBody & { type: 'psql' });
