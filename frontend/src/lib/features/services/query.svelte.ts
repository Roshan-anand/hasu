import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { ServiceListResponse } from './type';
import { getUserState } from '../global/store.svelte';

export const getGithubAppsQueryKey = (orgId: string) => ['github-apps', orgId] as const;
export const getOrgServicesQueryKey = (orgId: string) => ['services', 'org', orgId] as const;

export function useGetServicesQuery() {
	const { currentOrg } = getUserState();
	return createQuery(() => ({
		queryKey: ['services', currentOrg],
		queryFn: async () => {
			return api.get<ServiceListResponse>('/service').then((res) => res.data.services);
		}
	}));
}
