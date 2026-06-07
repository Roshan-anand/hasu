<script lang="ts">
	import AppSidebar from '@/components/app-sidebar.svelte';
	import * as Sidebar from '@/components/ui/sidebar/index.js';
	import ModeToggle from '@/components/mode-toggle.svelte';
	import { GetUserData } from '@/features/global/query';
	import { setBaseState } from '@/features/global/store.svelte';
	import AppBreadcrums from '@/components/app-breadcrums.svelte';
	import { setInstanceState } from '@/features/instance/context.svelte';

	let { children } = $props();

	const { org_id, org_name } = GetUserData();
	setBaseState(org_id, org_name);
	setInstanceState();
</script>

<Sidebar.Provider>
	<AppSidebar />
	<Sidebar.Inset>
		<header class="flex justify-between items-center p-2 border-b border-stroke">
			<Sidebar.Trigger />
			<AppBreadcrums />
			<ModeToggle />
		</header>
		<main class="p-2 flex-1 flex gap-5 flex-col">
			{@render children()}
		</main>
	</Sidebar.Inset>
</Sidebar.Provider>
