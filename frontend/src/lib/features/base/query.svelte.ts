import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import { GetUserData } from '../global/query';
import type { ApiRes } from '@/types';
import type { Organization } from '@/features/auth/type';
import type { OrphanVolume, ProjectListResponse } from './type';
import { getOrphanVolumesQueryKey, getOrgProjectsQueryKey, getOrgsQueryKey } from './const';

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

export function useGetAllOrgsQuery() {
	const { email } = GetUserData();
	return createQuery(() => ({
		queryKey: getOrgsQueryKey(email),
		queryFn: () => api.get<ApiRes<Organization[]>>('/org').then((res) => res.data.data),
		enabled: false
	}));
}

// Fetch orphan volumes filtered by predefined service type (e.g. "psql").
// Used during predef-db creation to let users reattach a compatible orphan volume.
export function useGetOrphanVolumesByTypeQuery(type: string) {
	const { org_id } = GetUserData();
	return createQuery(() => ({
		queryKey: ['orphan-volumes', 'org', org_id, 'type', type] as const,
		queryFn: async () => {
			const res = await api.get<ApiRes<OrphanVolume[]>>(`/volume/${type}`, {
				params: { org_id },
				validateStatus: (status) => (status >= 200 && status < 300) || status === 204
			});
			if (res.status === 204) return [];
			return res.data.data || [];
		},
		enabled: org_id !== '' && type !== ''
	}));
}
