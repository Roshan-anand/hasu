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
	DeletePsqlServicePayload,
	CreateServiceResponse
} from './type';
import type { ApiRes } from '@/types';
import { queryClient } from '@/query';
import { getInstanceServicesQueryKey } from './query.svelte';
import { getInstanceState } from '../instance/context.svelte';

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
			api.post<ApiRes<CreateServiceResponse>>('/service/app', payload).then((res) => res.data),
		onSuccess: ({ message }) => toast.success(message || 'App Service created successfully'),
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useCreatePsqlServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: CreatePsqlServicePayload) =>
			api.post<ApiRes<CreateServiceResponse>>('/service/psql', payload).then((res) => res.data),
		onSuccess: ({ message }, { volume }) => {
			// invalidate orphan volume caches when a reattach happened
			if (volume) {
				queryClient.invalidateQueries({ queryKey: ['orphan-volumes'] });
			}
			toast.success(message || 'PSQL Service created successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useDeleteAppServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: DeleteAppServicePayload) =>
			api.delete<ApiRes<null>>('/service/app', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			toast.success(message || 'Application deleted successfully');

			const instance = getInstanceState();
			if (!instance.id) return;
			queryClient.setQueryData(
				getInstanceServicesQueryKey(instance.id),
				(cachedRows: ServiceListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== service_id);
				}
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete Application')
	}));
}

export function useDeletePsqlServiceMutation() {
	const instance = getInstanceState();

	return createMutation(() => ({
		mutationFn: async (payload: DeletePsqlServicePayload) =>
			api.delete<ApiRes<null>>('/service/psql', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			queryClient.setQueryData(
				getInstanceServicesQueryKey(instance.id as string),
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
