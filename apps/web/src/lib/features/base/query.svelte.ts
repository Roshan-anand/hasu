import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { ApiRes } from '@/types';
import type { Organization } from '@/features/auth';
import type {
	OrphanVolume,
	ProjectListResponse,
	OrgProject,
	OrgVolume,
	GetAllInstancesResponse
} from './type';
import {
	getOrphanVolumesQueryKey,
	getOrgProjectsQueryKey,
	getOrgsQueryKey,
	getAllInstanceQueryKey
} from './const';
import type { AuthResponse } from '@/features/auth';
import { queryClient } from '@/query';
import { getOrgState } from './store.svelte';
import { getInstanceState } from '../instance';

const authUserQueryKey = () => ['auth', 'user'];

// query to fetch auth user data
export const fetchUserQuery = () =>
	queryClient.fetchQuery({
		queryKey: authUserQueryKey(),
		queryFn: () => api.get<ApiRes<AuthResponse>>('/auth/user').then((res) => res.data.data)
	});

// helper function to get auth user data from cache
export const GetUserData = (): AuthResponse =>
	queryClient.getQueryData<AuthResponse>(authUserQueryKey()) || {
		name: '',
		email: '',
		org_id: '',
		org_name: ''
	};

export const setUserData = (userData: AuthResponse | null) =>
	queryClient.setQueryData<AuthResponse | null>(authUserQueryKey(), userData);

export function useGetAllProjectsQuery() {
	return createQuery(() => {
		const currentOrg = getOrgState();
		return {
			queryKey: getOrgProjectsQueryKey(currentOrg.id),
			queryFn: async () =>
				api
					.get<ApiRes<ProjectListResponse>>('/project', {
						params: { org_id: currentOrg.id }
					})
					.then((res) => res.data.data),
			enabled: currentOrg.id !== ''
		};
	});
}

export function useGetOrphanVolumesQuery() {
	return createQuery(() => {
		const currentOrg = getOrgState();
		return {
			queryKey: getOrphanVolumesQueryKey(currentOrg.id),
			queryFn: async () =>
				await api
					.get<ApiRes<OrphanVolume[]>>('/volume', {
						params: { org_id: currentOrg.id }
					})
					.then((res) => res.data.data),
			enabled: currentOrg.id !== ''
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

export function useGetAllInstanceQuery(getProject: () => string) {
	return createQuery(() => {
		const { org_id } = GetUserData();
		const project = getProject();
		const instance = getInstanceState();

		return {
			queryKey: getAllInstanceQueryKey(project),
			queryFn: async () => {
				const res = await api.get<ApiRes<GetAllInstancesResponse>>('/instance', {
					params: { project, org_id }
				});

				instance.setInstances(res.data.data.instances);
				instance.setProjectID(res.data.data.project_id);
				return res.data.data;
			},
			enabled: project != '' && org_id !== ''
		};
	});
}

export function useGetOrgProjectsQuery(getOrgId: () => string) {
	return createQuery(() => {
		const orgId = getOrgId();
		return {
			queryKey: ['org-projects', orgId] as const,
			queryFn: async () =>
				api
					.get<ApiRes<OrgProject[]>>('/org/projects', {
						params: { org_id: orgId }
					})
					.then((res) => res.data.data),
			enabled: orgId !== ''
		};
	});
}

// Fetch orphan volumes for a given org (used in delete org confirmation UI)
export function useGetOrgVolumesQuery(getOrgId: () => string) {
	return createQuery(() => {
		const orgId = getOrgId();
		return {
			queryKey: ['org-volumes', orgId] as const,
			queryFn: async () =>
				api
					.get<ApiRes<OrgVolume[]>>('/org/volumes', {
						params: { org_id: orgId }
					})
					.then((res) => res.data.data),
			enabled: orgId !== ''
		};
	});
}

// Fetch orphan volumes filtered by predefined service type (e.g. "psql").
// Used during predef-db creation to let users reattach a compatible orphan volume.
export function useGetOrphanVolumesByTypeQuery(type: string) {
	return createQuery(() => {
		const currentOrg = getOrgState();
		return {
			queryKey: ['orphan-volumes', currentOrg.id, 'type', type] as const,
			queryFn: async () => {
				const res = await api.get<ApiRes<OrphanVolume[]>>(`/volume/${type}`, {
					params: { org_id: currentOrg.id },
					validateStatus: (status) => (status >= 200 && status < 300) || status === 204
				});
				if (res.status === 204) return [];
				return res.data.data || [];
			},
			enabled: currentOrg.id !== '' && type !== ''
		};
	});
}
