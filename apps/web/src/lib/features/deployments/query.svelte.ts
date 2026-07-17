import { api } from '@/axios';
import type { ApiRes, ServiceDeployment } from '@/types';
import { createQuery } from '@tanstack/svelte-query';

export const getDeploymentsQueryKey = (serviceId: string) =>
	['service-deployments', serviceId] as const;

export function useServiceDeploymentsQuery(getServiceId: () => string) {
	return createQuery(() => {
		const serviceId = getServiceId();
		return {
			queryKey: getDeploymentsQueryKey(serviceId),
			queryFn: async () => {
				return api
					.get<ApiRes<ServiceDeployment[]>>('/service/deployment', {
						params: { service_id: serviceId }
					})
					.then((res) => res.data.data);
			},
			enabled: serviceId !== ''
		};
	});
}
