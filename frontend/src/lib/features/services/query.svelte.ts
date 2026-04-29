import { api, axiosErr } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import { getServiceState } from './store.svelte';
import type { GithubApp } from './type';

export const getGithubAppsQueryKey = (orgId: string) => ['github-apps', orgId] as const;

export function useGithubAppsQuery(getOrgId: () => string) {
	const featureState = getServiceState();

	return createQuery(() => ({
		queryKey: getGithubAppsQueryKey(getOrgId()),
		queryFn: async () => {
			try {
				const response = await api.get<GithubApp[] | null>('/provider/github/app/list');
				const apps = response.data ?? [];
				featureState.githubApps = apps;
				return apps;
			} catch (error) {
				featureState.githubApps = [];
				const err = error instanceof Error ? error : new Error('Failed to load GitHub apps');
				axiosErr(err, 'Failed to load GitHub apps');
				return [];
			}
		},
		enabled: false
	}));
}
