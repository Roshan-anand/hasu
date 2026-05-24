import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { AppServiceDetails, GetBranchDomainRes, GetEnvRes, ServiceListResponse } from './type';
import type { ApiRes } from '@/types';

export const getProjectServicesQueryKey = (projectId: string) =>
	['services-list', 'project', projectId] as const;

export function useGetAllServicesQuery(getProjectId: () => string) {
	const projectId = getProjectId();
	return createQuery(() => ({
		queryKey: getProjectServicesQueryKey(projectId),
		queryFn: async () =>
			api
				.get<ApiRes<ServiceListResponse[]>>('/service', {
					params: { project_id: projectId }
				})
				.then((res) => res.data.data),
		enabled: projectId !== ''
	}));
}

export function useGetServiceDetailsQuery(getID: () => string) {
	const serviceId = getID();
	return createQuery(() => ({
		queryKey: ['service-details', serviceId],
		queryFn: async () =>
			api.get<ApiRes<AppServiceDetails>>(`/service/app/${serviceId}`).then((res) => res.data.data),
		enabled: serviceId !== ''
	}));
}

export function useGetBranchDomainQuery(getServiceId: () => string) {
	const serviceId = getServiceId();
	return createQuery(() => ({
		queryKey: ['branch-domain', serviceId],
		queryFn: async () =>
			api
				.get<ApiRes<GetBranchDomainRes>>('/service/app/domain', {
					params: { service_id: serviceId }
				})
				.then((res) => res.data.data),
		enabled: serviceId !== ''
	}));
}

export function useGetServiceEnvQuery(getServiceId: () => string) {
	const serviceId = getServiceId();
	return createQuery(() => ({
		queryKey: ['service-env', serviceId],
		queryFn: async () =>
			api
				.get<ApiRes<GetEnvRes>>('/service/app/env', {
					params: { service_id: serviceId }
				})
				.then((res) => res.data.data),
		enabled: serviceId !== ''
	}));
}
