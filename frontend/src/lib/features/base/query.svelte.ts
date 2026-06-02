import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import { GetUserData } from '../global/query';
import type { ApiRes } from '@/types';
import type { OrphanVolume, ProjectListResponse } from './type';
import { getOrphanVolumesQueryKey, getOrgProjectsQueryKey } from './const';

export function useGetAllProjectsQuery() {
	const { org_id } = GetUserData();
	return createQuery(() => ({
		queryKey: getOrgProjectsQueryKey(org_id),
		queryFn: async () =>
			api
				.get<ApiRes<ProjectListResponse[]>>('/project', {
					params: { org_id }
				})
				.then((res) => res.data.data),
		enabled: org_id !== ''
	}));
}

export function useGetOrphanVolumesQuery() {
	const { org_id } = GetUserData();
	return createQuery(() => {
		return {
			queryKey: getOrphanVolumesQueryKey(org_id),
			queryFn: async () => {
				const res = await api.get<ApiRes<OrphanVolume[]>>('/volume', {
					params: { org_id },
					validateStatus: (status) => (status >= 200 && status < 300) || status === 204
				});
				if (res.status === 204) return [];
				return res.data.data || [];
			},
			enabled: org_id !== ''
		};
	});
}
