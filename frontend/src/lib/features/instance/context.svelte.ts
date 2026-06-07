import { page } from '$app/state';
import { getContext, setContext } from 'svelte';

interface InstanceState {
	exp: RegExp;
	path: {
		project: string | null;
		service: string | null;
		type: string | null;
	};
	id: string | null;
	name: string;
	setCurrent: (id: string, name: string) => void;
}

class InstanceStateClass implements InstanceState {
	exp = /^(?!#\/(?:members|git|storage)?$).+$/;

	path = $derived.by(() => {
		if (!this.exp.test(page.url.hash))
			return {
				project: null,
				service: null,
				type: null
			};

		const paths = page.url.hash.split('/');
		paths.shift();
		return {
			project: paths[0],
			type: paths[1],
			service: paths[2]
		};
	});

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
