export type Project = {
	id: string;
	name: string;
	description: string;
};

export type CreateProjectPayload = {
	project_name: string;
	description: string;
};

export type DeleteProjectPayload = {
	id: string;
};

export type ApiMessageRes = {
	message: string;
};
