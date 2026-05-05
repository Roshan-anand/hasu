import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { ServiceListResponse } from './type';
import { getUserState } from '../global/store.svelte';

export const getOrgServicesQueryKey = (orgId: string) => ['services-list', 'org', orgId] as const;

export function useGetServicesQuery() {
	const { currentOrg } = getUserState();
	return createQuery(() => ({
		queryKey: getOrgServicesQueryKey(currentOrg.id),
		queryFn: async () => {
			return api.get<ServiceListResponse[]>('/service').then((res) => res.data);
		}
	}));
}
