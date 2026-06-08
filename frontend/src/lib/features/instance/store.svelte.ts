import { getContext, setContext } from 'svelte';
import type { Instance } from '../auth';

interface InstanceState {
	current: {
		id: string | null;
		name: string;
	};
	all: Instance[];
	setCurrent: (id: string, name: string) => void;
	setInstances: (instance: Instance[]) => void;
}

class InstanceStateClass implements InstanceState {
	private id: string | null = $state(null);
	private name: string = $state('');
	private instance: Instance[] = $state([]);

	get current() {
		return {
			id: this.id,
			name: this.name
		};
	}

	get all() {
		return this.instance;
	}

	setCurrent = (id: string, name: string) => {
		this.id = id;
		this.name = name;
	};

	setInstances = (instance: Instance[]) => {
		instance.forEach(({ is_production, id, name }) => {
			if (!is_production) return;
			this.id = id;
			this.name = name;
		});
		this.instance = instance;
	};
}

const DEFAULT_KEY = 'instance:state';

export const getInstanceState = (key: string = DEFAULT_KEY) =>
	getContext<InstanceState>(Symbol.for(key));

export const setInstanceState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new InstanceStateClass());
};
