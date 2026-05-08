import type { ServiceType } from '@/types.js';

export type ServiceTab = '' | 'deployment' | 'logs' | 'env';

export function load({ params, url }) {
	const queryString = url.hash.split('?')[1];
	const searchParams = new URLSearchParams(queryString);
	return {
		serviceType: params.service_type as ServiceType,
		serviceID: params.service_id,
		tab: searchParams.get('tab') as ServiceTab
	};
}
