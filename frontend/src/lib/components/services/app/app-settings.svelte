<script lang="ts">
	import { useGetAppServiceSettingsQuery } from '@/features/services';
	import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
	import { DotmSquare } from '@/components/loader';
	import InstancePRPreviewDropdown from '@/components/InstancePRPreviewDropdown.svelte';
	import { ServiceVisibility, ReplicaSelector } from './settings';
	import { AppDeletion } from '@/components/conformation';

	let { serviceID, serviceName }: { serviceName: string; serviceID: string } = $props();

	const settingsQuery = useGetAppServiceSettingsQuery(() => serviceID);
</script>

{#if settingsQuery.isPending}
	<div class="size-full flex items-center justify-center py-8">
		<DotmSquare size={40} dotSize={5} />
	</div>
{:else if settingsQuery.isError}
	<Card>
		<CardContent class="p-6">
			<p class="text-sm text-red-500">Failed to load service settings</p>
		</CardContent>
	</Card>
{:else if settingsQuery.data}
	{@const { domain, port, is_public, replicas } = settingsQuery.data}

	<ServiceVisibility
		{serviceID}
		initialDomain={domain}
		initialPort={port}
		initialIsPublic={is_public}
	/>

	<ReplicaSelector {serviceID} {replicas} />

	<Card>
		<CardHeader>
			<CardTitle class="text-lg">Actions</CardTitle>
		</CardHeader>
		<CardContent>
			<p class="text-sm text-muted-foreground mb-3">
				Create preview deployments from pull requests.
			</p>
			<InstancePRPreviewDropdown />
		</CardContent>
	</Card>
	<AppDeletion serviceId={serviceID} name={serviceName} />
{/if}
