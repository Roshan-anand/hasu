import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type {
	AppServiceDetails,
	GetEnvRes,
	PsqlServiceDetails,
	ServiceListResponse,
	PRInfo
} from './type';
import type { ApiRes } from '@/types';
import { getInstanceState } from '../instance';

export const getInstanceServicesQueryKey = (instanceID: string) =>
	['services-list', instanceID] as const;

export function useGetAllServicesQuery() {
	return createQuery(() => {
		const instance = getInstanceState();
		return {
			queryKey: getInstanceServicesQueryKey(instance.current.id as string),
			queryFn: async () =>
				api
					.get<ApiRes<ServiceListResponse[]>>('/service/all', {
						params: { instance_id: instance.current.id }
					})
					.then((res) => res.data.data),
			enabled: !!instance.current.id
		};
	});
}

export function useGetServiceIDQuery(getServiceName: () => string) {
	return createQuery(() => {
		const instance = getInstanceState();
		const serviceName = getServiceName();
		return {
			queryKey: getInstanceServicesQueryKey(instance.current.id as string),
			queryFn: async () =>
				api
					.get<ApiRes<string>>(`/service/${serviceName}`, {
						params: { instance_id: instance.current.id }
					})
					.then((res) => res.data.data),
			enabled: !!instance.current.id
		};
	});
}

export function useGetAppServiceDetailsQuery(getID: () => string) {
	return createQuery(() => {
		const serviceId = getID();
		return {
			queryKey: ['service-details', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<AppServiceDetails>>(`/service/app/${serviceId}`)
					.then((res) => res.data.data),
			enabled: serviceId !== ''
		};
	});
}

export function useGetServiceEnvQuery(getServiceId: () => string) {
	return createQuery(() => {
		const serviceId = getServiceId();
		return {
			queryKey: ['service-env', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<GetEnvRes>>('/service/app/env', {
						params: { service_id: serviceId }
					})
					.then((res) => res.data.data),
			enabled: serviceId !== ''
		};
	});
}

export function useGetPsqlServiceDetailsQuery(getID: () => string) {
	return createQuery(() => {
		const serviceId = getID();
		return {
			queryKey: ['psql-service-details', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<PsqlServiceDetails>>(`/service/psql/${serviceId}`)
					.then((res) => res.data.data),
			enabled: serviceId !== ''
		};
	});
}

export function useGetGithubPRListQuery(getServiceId: () => string) {
	return createQuery(() => {
		const serviceId = getServiceId();
		return {
			queryKey: ['github-prs', 'service', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<PRInfo[]>>('/provider/github/pr/list', {
						params: { service_id: serviceId }
					})
					.then((res) => res.data.data),
			enabled: serviceId !== ''
		};
	});
}

export function useGetGithubPRListByInstanceQuery() {
	return createQuery(() => {
		const instance = getInstanceState();
		return {
			queryKey: ['github-prs', 'instance', instance.current.id as string],
			queryFn: async () =>
				api
					.get<ApiRes<Record<string, PRInfo[]>>>('/provider/github/pr/instance', {
						params: { instance_id: instance.current.id }
					})
					.then((res) => res.data.data),
			enabled: !!instance.current.id
		};
	});
}
