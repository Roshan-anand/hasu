export type PsqlServiceName = 'psql';
export type AppServiceName = 'app';
export type ServiceType = PsqlServiceName | AppServiceName;

export type ServiceBase = {
	id: string;
	name: string;
	created_at: string;
};

export type PsqlService = {
	type: PsqlServiceName;
	db_name: string;
	db_user: string;
	db_password: string;
	image: string;
	internal_url: string;
};

export type AppService = {
	type: AppServiceName;
	git_repo_name: string;
	git_repo_url: string;
	git_branch: string;
	branch_name: string;
};

export type ServiceDeployment = {
	id: string;
	status: string;
	commit_msg: string;
	branch_name: string;
	created_at: string;
};
