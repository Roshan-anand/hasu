import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { ApiRes } from '@/types';
import type { Instance, Organization } from '@/features/auth/type';
import type { OrphanVolume, ProjectListResponse } from './type';
import { getOrphanVolumesQueryKey, getOrgProjectsQueryKey, getOrgsQueryKey } from './const';
import { getBaseState } from '../global/store.svelte';
import { GetUserData } from '../global/query';

export function useGetAllProjectsQuery() {
	return createQuery(() => {
		const base = getBaseState();
		return {
			queryKey: getOrgProjectsQueryKey(base.currentOrg.id),
			queryFn: async () =>
				api
					.get<ApiRes<ProjectListResponse>>('/project', {
						params: { org_id: base.currentOrg.id }
					})
					.then((res) => res.data.data),
			enabled: base.currentOrg.id !== ''
		};
	});
}

export function useGetOrphanVolumesQuery() {
	return createQuery(() => {
		const base = getBaseState();
		return {
			queryKey: getOrphanVolumesQueryKey(base.currentOrg.id),
			queryFn: async () => {
				const res = await api.get<ApiRes<OrphanVolume[]>>('/volume', {
					params: { org_id: base.currentOrg.id },
					validateStatus: (status) => (status >= 200 && status < 300) || status === 204
				});
				if (res.status === 204) return [];
				return res.data.data || [];
			},
			enabled: base.currentOrg.id !== ''
		};
	});
}

export function useGetAllOrgsQuery() {
	return createQuery(() => {
		const { email } = GetUserData();
		return {
			queryKey: getOrgsQueryKey(email),
			queryFn: () => api.get<ApiRes<Organization[]>>('/org').then((res) => res.data.data)
		};
	});
}

export function useGetAllInstanceQuery(getProject: () => string | null) {
	return createQuery(() => {
		const { org_id } = GetUserData();
		const project = getProject();

		return {
			queryKey: getOrgsQueryKey(project as string),
			queryFn: () =>
				api
					.get<ApiRes<Instance[]>>('/instance', {
						params: { project, org_id }
					})
					.then((res) => res.data.data),
			enabled: !!project && org_id !== ''
		};
	});
}

// Fetch orphan volumes filtered by predefined service type (e.g. "psql").
// Used during predef-db creation to let users reattach a compatible orphan volume.
export function useGetOrphanVolumesByTypeQuery(type: string) {
	return createQuery(() => {
		const base = getBaseState();
		return {
			queryKey: ['orphan-volumes', base.currentOrg.id, 'type', type] as const,
			queryFn: async () => {
				const res = await api.get<ApiRes<OrphanVolume[]>>(`/volume/${type}`, {
					params: { org_id: base.currentOrg.id },
					validateStatus: (status) => (status >= 200 && status < 300) || status === 204
				});
				if (res.status === 204) return [];
				return res.data.data || [];
			},
			enabled: base.currentOrg.id !== '' && type !== ''
		};
	});
}
