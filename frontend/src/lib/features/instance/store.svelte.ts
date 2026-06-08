import { getContext, setContext } from 'svelte';

interface InstanceState {
	id: string | null;
	name: string;
	setCurrent: (id: string, name: string) => void;
}

class InstanceStateClass implements InstanceState {
	id: string | null = $state(null);
	name: string = $state('');

	setCurrent = (id: string, name: string) => {
		this.id = id;
		this.name = name;
	};
}

const DEFAULT_KEY = 'instance:state';

export const getInstanceState = (key: string = DEFAULT_KEY) =>
	getContext<InstanceState>(Symbol.for(key));

export const setInstanceState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new InstanceStateClass());
};
