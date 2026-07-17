import { getContext, setContext } from 'svelte';
import type { Instance } from '../auth';

interface InstanceState {
	projectID: string;
	current: {
		id: string | null;
		name: string;
	};
	all: Instance[];
	setCurrent: (id: string, name: string) => void;
	setInstances: (instance: Instance[]) => void;
	setProjectID: (project_id: string) => void;
}

class InstanceStateClass implements InstanceState {
	private project_id: string = $state('');
	private id: string | null = $state(null);
	private name: string = $state('');
	private instance: Instance[] = $state([]);

	get projectID() {
		return this.project_id;
	}

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
		this.instance = instance.map((i) => {
			if (i.is_production) {
				this.id = i.id;
				this.name = i.name;
			}
			return i;
		});
	};

	setProjectID = (project_id: string) => {
		this.project_id = project_id;
	};
}

const DEFAULT_KEY = 'instance:state';

export const getInstanceState = (key: string = DEFAULT_KEY) =>
	getContext<InstanceState>(Symbol.for(key));

export const setInstanceState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new InstanceStateClass());
};
