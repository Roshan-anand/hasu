<script lang="ts">
	import { api } from '@/axios';
	import ServiceDetailApp from '@/components/services/service-detail-app.svelte';
	import ServiceDetailPsql from '@/components/services/service-detail-psql.svelte';
	import { Skeleton } from '@/components/ui/skeleton';
	import { setDeploymentsFeatureState } from '@/features/deployments/store.svelte';
	import { setServiceState } from '@/features/services/store.svelte';
	import type { ServiceDetails } from '@/types.js';
	import { createQuery } from '@tanstack/svelte-query';
	import Nav from './nav.svelte';
	import Deployment from './deployment.svelte';

	const { data } = $props();
	const serviceType = $derived(data.serviceType);
	const serviceId = $derived(data.serviceID);
	const tab = $derived(data.tab);
	setDeploymentsFeatureState();
	setServiceState();

	// query to fetch service details based on service type and id
	const serviceQuery = createQuery(() => ({
		queryKey: ['service-details', serviceType, serviceId],
		queryFn: async () => {
			const url =
				serviceType === 'app' ? `/service/app/${serviceId}` : `/service/psql/${serviceId}`;
			return api.get<ServiceDetails>(url).then((res) => res.data);
		},
		enabled: serviceId !== '' && (serviceType === 'psql' || serviceType === 'app')
	}));
</script>

<section class="p-2 flex-1">
	<div class="mb-2">
		<Nav {serviceType} {serviceId} {tab} />
	</div>

	{#if serviceType !== 'psql' && serviceType !== 'app'}
		<p class="text-muted-foreground">Invalid service type in URL</p>
	{:else if serviceId === ''}
		<p class="text-muted-foreground">Missing service id in URL</p>
	{:else if serviceQuery.isPending}
		<div class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 space-y-3">
			<Skeleton class="h-6 w-1/3" />
			<Skeleton class="h-4 w-2/3" />
			<Skeleton class="h-4 w-1/2" />
		</div>
	{:else if serviceQuery.isError || !serviceQuery.data}
		<p class="text-red-500">Failed to load service details</p>
	{:else if tab === 'logs'}
		<p class="text-muted-foreground">Logs tab content goes here</p>
	{:else if tab === 'deployment'}
		<Deployment {serviceId} />
	{:else if tab === 'env'}
		<p class="text-muted-foreground">Environment variables tab content goes here</p>
	{:else if serviceQuery.data.type === 'psql'}
		<ServiceDetailPsql service={serviceQuery.data} />
	{:else}
		<ServiceDetailApp service={serviceQuery.data} />
	{/if}
</section>
