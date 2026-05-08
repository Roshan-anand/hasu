import type { Organization } from '@/features/auth/type';
import { getContext, setContext } from 'svelte';

interface UserState {
	orgs: Organization[];
	setOrg: (orgs: Organization[]) => void;
	pushOrg: (orgs: Organization) => void;
}

class UserStateClass implements UserState {
	orgs = $state<Organization[]>([]);

	setOrg = (orgs: Organization[]) => (this.orgs = orgs);

	pushOrg = (newOrg: Organization) =>
		(this.orgs = [newOrg, ...this.orgs.filter((org) => org.id !== newOrg.id)]);
}

const DEFAULT_KEY = 'user:state';

export const getUserState = (key: string = DEFAULT_KEY) => getContext<UserState>(Symbol.for(key));

export const setUserState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new UserStateClass());
};
