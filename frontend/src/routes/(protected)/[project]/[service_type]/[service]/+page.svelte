<script lang="ts">
	import { useGetServiceIDQuery } from '@/features/services/query.svelte.js';
	import AppService from './app_service.svelte';
	import PsqlService from './psql_service.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	const { data } = $props();
	const { serviceName, projectName, serviceType, tab } = $derived(data);

	const getServiceID = useGetServiceIDQuery(() => serviceName);

	$effect(() => {
		if (getServiceID.isError)
			goto(
				resolve('/(protected)/[project]', {
					project: projectName
				})
			);
	});
</script>

{#if getServiceID.isPending}
	<p>loading....</p>
{:else if getServiceID.data}
	{@const serviceID = getServiceID.data}
	{#if serviceType === 'app'}
		<AppService {serviceName} {serviceID} {projectName} {tab} />
	{:else if serviceType === 'psql'}
		<PsqlService {serviceID} />
	{:else}
		<p class="text-muted-foreground">Invalid service type in URL</p>
	{/if}
{/if}
