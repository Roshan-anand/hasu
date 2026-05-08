import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { ServiceListResponse } from './type';
import { GetUserData } from '../global/query';

export const getOrgServicesQueryKey = (orgId: string) => ['services-list', 'org', orgId] as const;

export function useGetServicesQuery() {
	const { org_id } = GetUserData();
	return createQuery(() => ({
		queryKey: getOrgServicesQueryKey(org_id),
		queryFn: async () => {
			return api.get<ServiceListResponse[]>('/service').then((res) => res.data);
		}
	}));
}
