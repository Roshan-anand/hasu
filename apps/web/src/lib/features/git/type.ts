export type GithubApp = {
	name: string;
	app_id: number;
	created_at: string;
};

export type GitProvider = {
	name: string;
	icon: string;
	redirect: string;
};

export type DeleteGithubAppPayload = {
	app_id: number;
};
