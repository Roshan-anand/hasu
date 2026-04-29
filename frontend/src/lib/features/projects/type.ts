export interface Project {
	id: string;
	name: string;
	description: string;
}

export interface CreateProjectPayload {
	project_name: string;
	description: string;
}

export interface DeleteProjectPayload {
	id: string;
}

export interface ApiMessageRes {
	message: string;
}
