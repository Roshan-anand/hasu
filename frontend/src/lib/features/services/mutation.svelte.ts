import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type {
	ApiMessageRes,
	CreateServicePayload,
	DeleteServicePayload,
	GetReposPayload,
	GithubRepo,
	ServiceListResponse,
	UpdateBranchDomainPayload,
	UpdateEnvPayload
} from './type';
import { getOrgServicesQueryKey } from './query.svelte';
import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import type { ServiceType } from '@/types';
import { GetUserData } from '../global/query';
import { queryClient } from '@/query';

export function useGetReposMutation() {
	return createMutation(() => ({
		mutationFn: async ({ provider, appId }: GetReposPayload) =>
			api
				.get<GithubRepo[]>(provider.listApi, {
					params: { app_id: appId }
				})
				.then((res) => res.data),
		onError: (error) => axiosErr(error as Error, 'Failed to fetch repositories')
	}));
}

export function useCreateServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: CreateServicePayload) =>
			api.post<string>('/service/app', payload).then((res) => res.data),
		onSuccess: (service_id) => {
			toast.success('Service created successfully');
			goto(
				resolve('/(protected)/(core)/[service_type]/[service_id]?tab=deployment', {
					service_type: 'app' as ServiceType,
					service_id
				})
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useDeleteServiceMutation() {
	const { org_id } = GetUserData();

	return createMutation(() => ({
		mutationFn: async ({ service_id, type }: DeleteServicePayload) => {
			const url = type === 'psql' ? '/service/psql' : '/service/app';
			return api.delete<ApiMessageRes>(url, { data: { service_id } }).then((res) => res.data);
		},

		onSuccess: (response, payload) => {
			queryClient.setQueryData(
				getOrgServicesQueryKey(org_id),
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
			api.put<ApiMessageRes>('/service/app/domain', payload).then((res) => res.data),
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
			api.put<ApiMessageRes>('/service/app/env', payload).then((res) => res.data),
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

export function useRebuildServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: { branch_id: string }) =>
			api.put<string>('/service/app/rebuild', payload).then((res) => res.data),
		onSuccess: (service_id) => {
			toast.success('successfully rebuild the service');
			goto(
				resolve('/(protected)/(core)/[service_type]/[service_id]?tab=deployment', {
					service_type: 'app' as ServiceType,
					service_id
				})
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to rebuild the service')
	}));
}
