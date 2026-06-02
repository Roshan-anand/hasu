import type { Organization } from '@/features/auth/type';
import { getContext, setContext } from 'svelte';

interface UserState {
	currentOrg: Organization;
	setOrg: (org: Organization) => void;
}

class UserStateClass implements UserState {
	currentOrg = $state<Organization>({ name: '', id: '' });

	setOrg = (org: Organization) => (this.currentOrg = org);
}

const DEFAULT_KEY = 'org:state';

export const getOrgState = (key: string = DEFAULT_KEY) => getContext<UserState>(Symbol.for(key));

export const setOrgState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new UserStateClass());
};
