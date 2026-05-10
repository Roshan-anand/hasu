<script lang="ts">
	import AppDeployments from '@/components/services/app/app-deployments.svelte';
	import AppHome from '@/components/services/app/app-home.svelte';
	import { resolve } from '$app/paths';
	import * as NavigationMenu from '@/components/ui/navigation-menu';
	import { NavItems } from '@/features/services/const';
	import type { ServiceTab } from '@/features/services/type';

	const { serviceId, tab }: { serviceId: string; tab: ServiceTab } = $props();
</script>

<section class="p-2 flex-1">
	<div class="mb-2">
		<NavigationMenu.Root viewport={false} class="w-full max-w-full">
			<NavigationMenu.List class="flex-1 w-full rounded-lg bg-card p-1">
				{#each NavItems as item (item.label)}
					<NavigationMenu.Item>
						<NavigationMenu.Link
							href={resolve(`/(protected)/(core)/[service_type]/[service_id]?tab=${item.tab}`, {
								service_type: 'app',
								service_id: serviceId
							})}
							data-active={tab == item.tab || (tab == undefined && item.tab == '')}
							class="cursor-pointer px-3 py-2"
						>
							{item.label}
						</NavigationMenu.Link>
					</NavigationMenu.Item>
				{/each}
			</NavigationMenu.List>
		</NavigationMenu.Root>
	</div>

	{#if serviceId === ''}
		<p class="text-muted-foreground">Missing service id in URL</p>
	{:else if tab === 'deployment'}
		<AppDeployments {serviceId} />
	{:else if tab === 'env'}
		<p class="text-muted-foreground">Environment variables tab content goes here</p>
	{:else}
		<AppHome {serviceId} />
	{/if}
</section>
