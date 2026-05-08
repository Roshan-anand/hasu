<script lang="ts">
	import { resolve } from '$app/paths';
	import * as NavigationMenu from '@/components/ui/navigation-menu';
	import type { ServiceType } from '@/types.js';
	import type { ServiceTab } from './+page.js';

	type NavItem = {
		label: string;
		tab: ServiceTab;
	};

	const navItems: NavItem[] = [
		{ label: 'General', tab: '' },
		{ label: 'Deployment', tab: 'deployment' },
		{ label: 'Environment', tab: 'env' },
		{ label: 'Logs', tab: 'logs' }
	];

	let {
		serviceType,
		serviceId,
		tab
	}: { serviceType: ServiceType; serviceId: string; tab: ServiceTab } = $props();

	function isActive(nextTab: ServiceTab | '') {
		return tab === nextTab;
	}
</script>

<NavigationMenu.Root viewport={false} class="max-w-none w-full">
	<NavigationMenu.List class="w-full justify-start rounded-lg border bg-card p-1">
		{#each navItems as item (item.label)}
			<NavigationMenu.Item>
				<NavigationMenu.Link
					href={resolve(`/(protected)/(core)/[service_type]/[service_id]?tab=${item.tab}`, {
						service_type: serviceType,
						service_id: serviceId
					})}
					data-active={isActive(item.tab) ? '' : undefined}
					class="cursor-pointer px-3 py-2"
				>
					{item.label}
				</NavigationMenu.Link>
			</NavigationMenu.Item>
		{/each}
	</NavigationMenu.List>
</NavigationMenu.Root>
