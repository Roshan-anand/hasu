import type { AuthResponse, Organization } from '@/features/auth/type';
import { getContext, setContext } from 'svelte';

interface UserState {
	name: string;
	email: string;
	currentOrg: Organization;
	orgs: Organization[];
	isAuth: boolean;
	setCurrentOrg: (org: Organization) => void;
	setOrg: (orgs: Organization[]) => void;
	pushOrg: (orgs: Organization) => void;
	setUser: (data: AuthResponse) => void;
}

class UserStateClass implements UserState {
	name = $state('');
	email = $state('');
	currentOrg = $state<Organization>({ id: '', name: '' });
	orgs = $state<Organization[]>([]);
	isAuth = $state(false);

	setCurrentOrg = (org: Organization) => (this.currentOrg = org);

	setOrg = (orgs: Organization[]) => (this.orgs = orgs);

	pushOrg = (newOrg: Organization) =>
		(this.orgs = [newOrg, ...this.orgs.filter((org) => org.id !== newOrg.id)]);

	setUser = (data: AuthResponse) => {
		this.name = data.name;
		this.email = data.email;
		this.currentOrg = {
			id: data.org_id,
			name: data.org_name
		};
		this.isAuth = true;
	};
}

const DEFAULT_KEY = 'user:state';

export const getUserState = (key: string = DEFAULT_KEY) => {
	return getContext<UserState>(Symbol.for(key));
};

export const setUserState = (key: string = DEFAULT_KEY) => {
	const userState = new UserStateClass();
	setContext(Symbol.for(key), userState);
};
