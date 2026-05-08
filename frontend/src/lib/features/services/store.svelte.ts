import { getContext, setContext } from 'svelte';
import type { GithubApp, GithubRepo } from './type';

export interface ServiceStore {
	githubApps: GithubApp[];
	githubRepos: GithubRepo[];
	setGithubApps: (apps: GithubApp[]) => void;
}

class ServiceStoreClass implements ServiceStore {
	githubApps = $state<GithubApp[]>([]);
	githubRepos = $state<GithubRepo[]>([]);
	setGithubApps = (apps: GithubApp[]) => (this.githubApps = apps);
}

const DEFAULT_KEY = 'services:feature:state';

export const getServiceState = (key: string = DEFAULT_KEY) =>
	getContext<ServiceStore>(Symbol.for(key));

export const setServiceState = (key: string = DEFAULT_KEY) =>
	setContext(Symbol.for(key), new ServiceStoreClass());
