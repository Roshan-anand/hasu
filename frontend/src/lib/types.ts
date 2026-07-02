export type PsqlServiceName = 'psql';
export type RedisServiceName = 'redis';
export type AppServiceName = 'app';
export type ServiceType = PsqlServiceName | RedisServiceName | AppServiceName;

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
	is_current: boolean;
	status: string;
	commit_hash: string;
	commit_msg: string;
	branch_name: string;
	created_at: string;
};

export type ApiRes<T> = {
	message: string;
	data: T;
};
