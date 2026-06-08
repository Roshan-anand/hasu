import { getContext, setContext } from 'svelte';

interface OrgState {
	id: string;
	name: string;
	setCurrentOrg: (id: string, name: string) => void;
}

class OrgStateClass implements OrgState {
	constructor(id: string, name: string) {
		this.id = id;
		this.name = name;
	}

	id: string = $state('');
	name: string = $state('');

	setCurrentOrg = (id: string, name: string) => {
		this.id = id;
		this.name = name;
	};
}

const DEFAULT_KEY = 'org:state';

export const getOrgState = (key: string = DEFAULT_KEY) => getContext<OrgState>(Symbol.for(key));

export const setOrgState = (id: string, name: string, key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new OrgStateClass(id, name));
};
