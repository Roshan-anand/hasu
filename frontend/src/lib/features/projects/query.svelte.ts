import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { Project } from './type';

export const getProjectsQueryKey = (orgId: string) => ['projects', orgId] as const;

export const getServiceCreateProjectsQueryKey = (orgId: string) =>
	['projects', orgId, 'service-create'] as const;

export function useProjectsQuery(getOrgId: () => string) {
	return createQuery(() => ({
		queryKey: getProjectsQueryKey(getOrgId()),
		queryFn: async () => api.get<Project[]>('/project/all').then((res) => res.data),
		enabled: getOrgId() !== ''
	}));
}

export function useServiceCreateProjectsQuery(
	getOrgId: () => string,
	getIsProjectScoped: () => boolean
) {
	return createQuery(() => ({
		queryKey: getServiceCreateProjectsQueryKey(getOrgId()),
		queryFn: async () => api.get<Project[]>('/project/all').then((res) => res.data),
		enabled: !getIsProjectScoped() && getOrgId() !== ''
	}));
}
