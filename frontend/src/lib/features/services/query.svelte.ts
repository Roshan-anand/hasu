import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type {
	AppServiceDetails,
	GetBranchDomainRes,
	GetEnvRes,
	PsqlServiceDetails,
	ServiceListResponse
} from './type';
import type { ApiRes } from '@/types';
import { getInstanceState } from '../instance';

export const getInstanceServicesQueryKey = (instanceID: string) =>
	['services-list', instanceID] as const;

export function useGetAllServicesQuery() {
	return createQuery(() => {
		const instance = getInstanceState();
		return {
			queryKey: getInstanceServicesQueryKey(instance.id as string),
			queryFn: async () =>
				api
					.get<ApiRes<ServiceListResponse[]>>('/service/all', {
						params: { instance_id: instance.id }
					})
					.then((res) => res.data.data),
			enabled: !!instance.id
		};
	});
}

export function useGetServiceIDQuery(getServiceName: () => string) {
	return createQuery(() => {
		const instance = getInstanceState();
		const serviceName = getServiceName();
		return {
			queryKey: getInstanceServicesQueryKey(instance.id as string),
			queryFn: async () =>
				api
					.get<ApiRes<string>>(`/service/${serviceName}`, {
						params: { instance_id: instance.id }
					})
					.then((res) => res.data.data),
			enabled: !!instance.id
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

export function useGetBranchDomainQuery(getServiceId: () => string) {
	return createQuery(() => {
		const serviceId = getServiceId();
		return {
			queryKey: ['branch-domain', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<GetBranchDomainRes>>('/service/app/domain', {
						params: { service_id: serviceId }
					})
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
