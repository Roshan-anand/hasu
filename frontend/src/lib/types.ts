export type ServiceType = 'psql' | 'app';

export type ServiceBase = {
	id: string;
	project_id: string;
	type: ServiceType;
	service_id: string;
	name: string;
	app_name: string;
	description: string;
	created_at: string;
};

export type PsqlService = {
	type: 'psql';
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
	internal_url: string;
};

export type AppService = {
	type: 'app';
	git_provider: 'github' | 'gitlab' | 'bitbucket';
	gh_app_id: number;
	git_repo_id: string;
	git_repo_name: string;
	git_repo_url: string;
	git_branch: string;
	build_path: string;
	watch_path: string;
};

export type ServiceDetails = ServiceBase & (PsqlService | AppService);

export type ServiceDeployment = {
	id: string;
	service_id: string;
	name: string;
	status: string;
	created_at: string;
};
