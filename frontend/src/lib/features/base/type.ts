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

export type RenameOrgPayload = {
	org_id: string;
	name: string;
};

export type DeleteOrgPayload = {
	org_id: string;
};

export type TransferProjectPayload = {
	project_id: string;
	target_org_id: string;
};

export type OrgProject = {
	id: string;
	name: string;
	created_at: string;
};

export type RenameProjectPayload = {
	project_id: string;
	org_id: string;
	name: string;
};

export type RenameProjectResponse = {
	id: string;
	name: string;
	created_at: string;
};

export type RenameInstancePayload = {
	instance_id: string;
	project_id: string;
	name: string;
};
