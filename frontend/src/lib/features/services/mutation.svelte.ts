import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type {
	CreateServicePayload,
	DeleteServicePayload,
	GetReposPayload,
	GithubRepo,
	ServiceListResponse,
	UpdateBranchDomainPayload,
	UpdateEnvPayload
} from './type';
import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import type { ApiRes, ServiceType } from '@/types';
import { getProjectServicesQueryKey } from './query.svelte';
import { queryClient } from '@/query';

export function useGetReposMutation() {
	return createMutation(() => ({
		mutationFn: async ({ provider, appId }: GetReposPayload) =>
			api
				.get<ApiRes<GithubRepo[]>>(provider.listApi, {
					params: { app_id: appId }
				})
				.then((res) => res.data.data),
		onError: (error) => axiosErr(error as Error, 'Failed to fetch repositories')
	}));
}

export function useCreateServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: CreateServicePayload) =>
			api.post<ApiRes<string>>('/service/app', payload).then((res) => res.data),
		onSuccess: (res) => {
			toast.success('Service created successfully');
			goto(
				resolve('/(protected)/(core)/[service_type]/[service_id]?tab=deployment', {
					service_type: 'app' as ServiceType,
					service_id: res.data
				})
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useDeleteServiceMutation(getProjectId: () => string) {
	const projectId = getProjectId();
	return createMutation(() => ({
		mutationFn: async ({ service_id, type }: DeleteServicePayload) => {
			const url = type === 'psql' ? '/service/psql' : '/service/app';
			return api.delete<ApiRes<null>>(url, { data: { service_id } }).then((res) => res.data);
		},

		onSuccess: (response, payload) => {
			queryClient.setQueryData(
				getProjectServicesQueryKey(projectId),
				(cachedRows: ServiceListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== payload.service_id);
				}
			);
			toast.success(response.message || 'Service deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete service')
	}));
}

export function useUpdateBranchDomainMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdateBranchDomainPayload) =>
			api.put<ApiRes<null>>('/service/app/domain', payload).then((res) => res.data),
		onSuccess: (response) => {
			queryClient.invalidateQueries({
				queryKey: ['branch-domain', getServiceId()]
			});
			toast.success(response.message || 'Domain updated successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update domain')
	}));
}

export function useUpdateEnvMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdateEnvPayload) =>
			api.put<ApiRes<null>>('/service/app/env', payload).then((res) => res.data),
		onSuccess: (response) => {
			queryClient.invalidateQueries({
				queryKey: ['service-env', getServiceId()]
			});
			// TODO : show a button to rebuild / restart the service
			toast.success(response.message || 'Env updated successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update env')
	}));
}
