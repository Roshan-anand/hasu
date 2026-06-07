import { getContext, setContext } from 'svelte';
import type { Organization } from '../auth/type';

interface BaseState {
	currentOrg: Organization;
	setCurrentOrg: (id: string, name: string) => void;
}

class BaseStateClass implements BaseState {
	constructor(id: string, name: string) {
		this.currentOrg = { id, name };
	}

	currentOrg: Organization = $state({ id: '', name: '' });

	setCurrentOrg = (id: string, name: string) => {
		this.currentOrg = { id, name };
	};
}

const DEFAULT_KEY = 'org:state';

export const getBaseState = (key: string = DEFAULT_KEY) => getContext<BaseState>(Symbol.for(key));

export const setBaseState = (id: string, name: string, key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new BaseStateClass(id, name));
};
