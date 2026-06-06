export type LoginPayload = {
	email: string;
	password: string;
};

export type RegisterPayload = {
	name: string;
	email: string;
	password: string;
	org_name: string;
};

export type Organization = {
	id: string;
	name: string;
};

export type Instance = {
	id: string;
	name: string;
	is_production: boolean;
};

export type AuthResponse = {
	name: string;
	email: string;
	org_id: string;
	org_name: string;
};
