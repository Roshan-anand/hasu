<script lang="ts">
	import AppService from './app_service.svelte';
	import PsqlService from './psql_service.svelte';

	const { data } = $props();
	const serviceType = $derived(data.serviceType);
	const serviceId = $derived(data.serviceID);
	const tab = $derived(data.tab);
</script>

{#if serviceType === 'app'}
	<AppService {serviceId} {tab} />
{:else if serviceType === 'psql'}
	<PsqlService {serviceId} />
{:else}
	<p class="text-muted-foreground">Invalid service type in URL</p>
{/if}
