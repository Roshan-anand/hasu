import { api } from '@/axios';
import type { ServiceDeployment } from '@/types';
import { createQuery } from '@tanstack/svelte-query';

export const getDeploymentsQueryKey = (serviceId: string) =>
	['service-deployments', serviceId] as const;

export function useServiceDeploymentsQuery(getServiceId: () => string) {
	return createQuery(() => ({
		queryKey: getDeploymentsQueryKey(getServiceId()),
		queryFn: async () => {
			return api
				.get<ServiceDeployment[]>('/service/deployment', {
					params: { service_id: getServiceId() }
				})
				.then((res) => res.data);
		},
		enabled: getServiceId() !== ''
	}));
}
