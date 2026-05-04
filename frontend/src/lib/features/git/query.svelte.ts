import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { GitProvider, GithubApp } from './type';
import { getUserState } from '../global/store.svelte';

export const gitProviders: GitProvider[] = [
	{
		name: 'Github',
		icon: 'meteor-icons:github',
		redirect: '/api/provider/github/app/create'
	},
	{
		name: 'GitLab',
		icon: 'material-icon-theme:gitlab',
		redirect: ''
	},
	{
		name: 'BitBucket',
		icon: 'material-icon-theme:bitbucket',
		redirect: ''
	}
];

export function useGithubAppsQuery() {
	const { currentOrg } = getUserState();
	return createQuery(() => ({
		queryKey: ['github-apps', currentOrg],
		queryFn: () => api.get<GithubApp[] | null>('/provider/github/app/list').then((res) => res.data),
		enabled: currentOrg.id != ''
	}));
}
