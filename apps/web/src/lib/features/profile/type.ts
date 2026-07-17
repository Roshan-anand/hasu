export type ProfileResponse = {
	id: string;
	name: string;
	email: string;
	role: string;
	avatar: string;
	created_at: string;
};

export type UpdateProfilePayload = {
	name: string;
	email: string;
	avatar: string;
};

export type ChangePasswordPayload = {
	old_password: string;
	new_password: string;
};
