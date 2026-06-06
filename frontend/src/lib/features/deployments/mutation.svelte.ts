import { api, axiosErr } from '@/axios';
import { queryClient } from '@/query';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import { getDeploymentsQueryKey } from './query.svelte';
import type { DeleteDeploymentPayload } from './type';
import type { ApiRes, ServiceDeployment } from '@/types';

export function useDeleteDeploymentMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async ({ deployment_id }: DeleteDeploymentPayload) => {
			return api
				.delete<ApiRes<null>>('/service/deployment', {
					data: { deployment_id }
				})
				.then((res) => res.data);
		},
		onSuccess: ({ message }, { deployment_id }) => {
			queryClient.setQueryData(
				getDeploymentsQueryKey(getServiceId()),
				(cachedRows: ServiceDeployment[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== deployment_id);
				}
			);
			toast.success(message || 'Deployment deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete deployment')
	}));
}

export function useRebuildServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: { service_id: string }) =>
			api.post<ApiRes<string>>('/service/app/rebuild', payload).then((res) => res.data),
		onSuccess: () => toast.success('successfully rebuild the service'),
		onError: (error) => axiosErr(error as Error, 'Failed to rebuild the service')
	}));
}

export function useRollbackServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: { service_id: string }) =>
			api.post<ApiRes<null>>('/service/app/rollback', payload).then((res) => res.data),
		onSuccess: ({ message }) => toast.success(message || 'successfully started rollback'),
		onError: (error) => axiosErr(error as Error, 'Failed to start rollback')
	}));
}
