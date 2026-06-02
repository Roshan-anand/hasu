import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type {
	CreateServicePayload,
	GetReposPayload,
	GithubRepo,
	CreatePsqlServicePayload,
	RedeployPsqlServicePayload,
	ServiceListResponse,
	UpdateBranchDomainPayload,
	UpdateEnvPayload,
	UpdatePsqlServicePayload,
	DeleteAppServicePayload,
	DeletePsqlServicePayload
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
		onSuccess: ({ data, message }) => {
			toast.success(message || 'App Service created successfully');
			goto(
				resolve('/(protected)/(core)/[service_type]/[service_id]?tab=deployment', {
					service_type: 'app' as ServiceType,
					service_id: data
				})
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useCreatePsqlServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: CreatePsqlServicePayload) =>
			api.post<ApiRes<string>>('/service/psql', payload).then((res) => res.data),
		onSuccess: ({ message }, { project_id }) => {
			toast.success(message || 'PSQL Service created successfully');
			goto(
				resolve('/(protected)/(core)/project/[project_id]', {
					project_id
				})
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useDeleteAppServiceMutation(getProjectId: () => string) {
	const projectId = getProjectId();
	return createMutation(() => ({
		mutationFn: async (payload: DeleteAppServicePayload) =>
			api.delete<ApiRes<null>>('/service/app', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			queryClient.setQueryData(
				getProjectServicesQueryKey(projectId),
				(cachedRows: ServiceListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== service_id);
				}
			);
			toast.success(message || 'Application deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete Application')
	}));
}

export function useDeletePsqlServiceMutation(getProjectId: () => string) {
	const projectId = getProjectId();
	return createMutation(() => ({
		mutationFn: async (payload: DeletePsqlServicePayload) =>
			api.delete<ApiRes<null>>('/service/psql', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			queryClient.setQueryData(
				getProjectServicesQueryKey(projectId),
				(cachedRows: ServiceListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== service_id);
				}
			);
			toast.success(message || 'psql deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete psql')
	}));
}

export function useUpdateBranchDomainMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdateBranchDomainPayload) =>
			api.put<ApiRes<null>>('/service/app/domain', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['branch-domain', getServiceId()]
			});
			toast.success(message || 'Domain updated successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update domain')
	}));
}

export function useUpdateEnvMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdateEnvPayload) =>
			api.put<ApiRes<null>>('/service/app/env', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['service-env', getServiceId()]
			});
			// TODO : show a button to rebuild / restart the service
			toast.success(message || 'Env updated successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update env')
	}));
}

export function useUpdatePsqlServiceMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdatePsqlServicePayload) =>
			api.put<ApiRes<null>>('/service/psql', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['psql-service-details', getServiceId()]
			});
			toast.success(message || 'PSQL details updated successfully');
			// TODO : show an info msg to redploy
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update PSQL details')
	}));
}

export function useRedeployPsqlServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: RedeployPsqlServicePayload) =>
			api.post<ApiRes<null>>('/service/psql/redeploy', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			toast.success(message || 'PSQL redeploy started');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to redeploy PSQL service')
	}));
}
