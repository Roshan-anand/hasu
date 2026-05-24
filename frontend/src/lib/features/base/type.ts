// Project types map backend project responses for UI usage.
export type ProjectListResponse = {
	id: string;
	organization_id: string;
	name: string;
	network_name: string;
	created_at: string;
};

export type CreateProjectPayload = {
	name: string;
};

export type DeleteProjectPayload = {
	project_id: string;
};
