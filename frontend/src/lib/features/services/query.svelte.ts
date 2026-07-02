import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type {
	AppServiceDetails,
	AppServiceSettings,
	GetEnvRes,
	PsqlServiceDetails,
	RedisServiceDetails,
	ServiceListResponse,
	PRInfo,
	ServiceDependency,
	DependencyTarget,
	DependencyGraphResponse
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

export function useGetServiceDependenciesQuery(getServiceId: () => string) {
	return createQuery(() => {
		const serviceId = getServiceId();
		return {
			queryKey: ['service-dependencies', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<{ dependencies: ServiceDependency[] }>>('/service/app/dependencies', {
						params: { service_id: serviceId }
					})
					.then((res) => res.data.data.dependencies),
			enabled: serviceId !== ''
		};
	});
}

export function useGetDependencyTargetsQuery(getServiceId: () => string) {
	return createQuery(() => {
		const serviceId = getServiceId();
		return {
			queryKey: ['dependency-targets', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<{ targets: DependencyTarget[] }>>('/service/app/dependency-targets', {
						params: { service_id: serviceId }
					})
					.then((res) => res.data.data.targets),
			enabled: serviceId !== ''
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

export function useGetRedisServiceDetailsQuery(getID: () => string) {
	return createQuery(() => {
		const serviceId = getID();
		return {
			queryKey: ['redis-service-details', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<RedisServiceDetails>>(`/service/redis/${serviceId}`)
					.then((res) => res.data.data),
			enabled: serviceId !== ''
		};
	});
}

export function useGetAppServiceSettingsQuery(getID: () => string) {
	return createQuery(() => {
		const serviceId = getID();
		return {
			queryKey: ['service-settings', serviceId],
			queryFn: async () =>
				api
					.get<ApiRes<AppServiceSettings>>('/service/app/settings', {
						params: { service_id: serviceId }
					})
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

export function useGetDependencyGraphQuery() {
	return createQuery(() => {
		const instance = getInstanceState();
		return {
			queryKey: ['dependency-graph', instance.current.id as string],
			queryFn: async () =>
				api
					.get<ApiRes<DependencyGraphResponse>>(`/instance/${instance.current.id}/dependency-graph`)
					.then((res) => res.data.data),
			enabled: !!instance.current.id
		};
	});
}
