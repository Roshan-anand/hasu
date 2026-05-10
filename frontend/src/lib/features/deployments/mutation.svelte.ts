import { api, axiosErr } from '@/axios';
import { queryClient } from '@/query';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import { getDeploymentsQueryKey } from './query.svelte';
import type { DeleteDeploymentPayload, DeleteDeploymentResponse } from './type';
import type { ServiceDeployment } from '@/types';

export function useDeleteDeploymentMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async ({ deployment_id }: DeleteDeploymentPayload) => {
			return api
				.delete<DeleteDeploymentResponse>('/service/deployment', {
					data: { deployment_id }
				})
				.then((res) => res.data);
		},
		onSuccess: (response, payload) => {
			queryClient.setQueryData(
				getDeploymentsQueryKey(getServiceId()),
				(cachedRows: ServiceDeployment[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== payload.deployment_id);
				}
			);
			toast.success(response.message || 'Deployment deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete deployment')
	}));
}
