export type ProjectResponse = {
	id: string;
	name: string;
	created_at: string;
};

export type ProjectListResponse = ProjectResponse[];

export type CreateProjectPayload = {
	name: string;
};

export type DeleteProjectPayload = {
	project_id: string;
	volumes: string[];
};

export type DeleteVolumePayload = {
	volumes: string[];
};

export type OrphanVolume = {
	id: string;
	volume: string;
	type: string;
	created_at: string;
};

export type SwitchOrgPayload = {
	org_id: string;
};

export type CreateOrgPayload = {
	name: string;
};
