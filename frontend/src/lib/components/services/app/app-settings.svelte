<script lang="ts">
	import { useGetAppServiceSettingsQuery } from '@/features/services';
	import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
	import { DotmSquare } from '@/components/loader';
	import InstancePRPreviewDropdown from '@/components/InstancePRPreviewDropdown.svelte';
	import { ServiceVisibility, ReplicaSelector, BuildSettings } from './settings';
	import type { PRInfo } from '@/features/services';

	let { serviceID }: { serviceID: string } = $props();

	const settingsQuery = useGetAppServiceSettingsQuery(() => serviceID);

	function handlePRSelect(_serviceName: string, _pr: PRInfo) {
		// preview creation handled by the InstancePRPreviewDropdown internally
	}
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
	{@const {
		domain,
		port,
		is_public,
		replicas,
		build_path,
		watch_path,
		docker_filepath,
		docker_contextpath,
		docker_buildstage
	} = settingsQuery.data}

	<ServiceVisibility
		{serviceID}
		initialDomain={domain}
		initialPort={port}
		initialIsPublic={is_public}
	/>

	<ReplicaSelector {serviceID} {replicas} />

	<BuildSettings
		{serviceID}
		initialBuildPath={build_path}
		initialWatchPath={watch_path}
		initialDockerFilepath={docker_filepath}
		initialDockerContextpath={docker_contextpath}
		initialDockerBuildstage={docker_buildstage}
	/>

	<Card>
		<CardHeader>
			<CardTitle class="text-lg">Actions</CardTitle>
		</CardHeader>
		<CardContent>
			<p class="text-sm text-muted-foreground mb-3">
				Create preview deployments from pull requests.
			</p>
			<InstancePRPreviewDropdown onSelect={handlePRSelect} />
		</CardContent>
	</Card>
{/if}
