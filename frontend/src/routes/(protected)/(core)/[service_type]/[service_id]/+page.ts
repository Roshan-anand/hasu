import type { ServiceTab } from '@/features/services/type';
import type { ServiceType } from '@/types.js';

export function load({ params, url }) {
	const queryString = url.hash.split('?')[1];
	const searchParams = new URLSearchParams(queryString);
	return {
		serviceType: params.service_type as ServiceType,
		serviceID: params.service_id,
		tab: searchParams.get('tab') as ServiceTab
	};
}
