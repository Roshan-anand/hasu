import { getContext, setContext } from 'svelte';
import type { CreateServiceResponse, GithubApp, GithubRepo } from './type';

export interface ServiceStore {
	githubApps: GithubApp[];
	githubRepos: GithubRepo[];
	searchQuery: string;
	createDialogOpen: boolean;
	openCreateDialog: () => void;
	closeCreateDialog: () => void;
	setGithubApps: (apps: GithubApp[]) => void;
	afterCreateSuccess: (response: CreateServiceResponse) => Promise<void>;
	setAfterCreateSuccess: (fn: (response: CreateServiceResponse) => Promise<void>) => void;
}

class ServiceStoreClass implements ServiceStore {
	// AI-generated: consolidate service UI/feature state into one context class.
	githubApps = $state<GithubApp[]>([]);
	githubRepos = $state<GithubRepo[]>([]);
	searchQuery = $state('');
	createDialogOpen = $state(false);
	setGithubApps = (apps: GithubApp[]) => (this.githubApps = apps);
	openCreateDialog = () => {
		this.createDialogOpen = true;
	};
	closeCreateDialog = () => {
		this.createDialogOpen = false;
	};

	afterCreateSuccess = $state<(response: CreateServiceResponse) => Promise<void>>(async () => {});

	setAfterCreateSuccess = (fn: (response: CreateServiceResponse) => Promise<void>) => {
		this.afterCreateSuccess = fn;
	};
}

const DEFAULT_KEY = 'services:feature:state';

export const getServiceState = (key: string = DEFAULT_KEY) => {
	return getContext<ServiceStore>(Symbol.for(key));
};

export const setServiceState = (key: string = DEFAULT_KEY) => {
	const state = new ServiceStoreClass();
	setContext(Symbol.for(key), state);
	return state;
};
