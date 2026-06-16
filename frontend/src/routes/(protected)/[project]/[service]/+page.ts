import type { ServiceTab } from '@/features/services';

export function load({ params, url }) {
	const queryString = url.hash.split('?')[1];
	const searchParams = new URLSearchParams(queryString);

	return {
		serviceName: params.service,
		projectName: params.project,
		tab: searchParams.get('tab') as ServiceTab
	};
}
