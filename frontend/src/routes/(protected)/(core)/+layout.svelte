<script lang="ts">
	import AppSidebar from '@/components/app-sidebar.svelte';
	import * as Sidebar from '@/components/ui/sidebar/index.js';
	import ModeToggle from '@/components/mode-toggle.svelte';

	let { children } = $props();
</script>

<Sidebar.Provider>
	<AppSidebar />
	<Sidebar.Inset>
		<header class="flex justify-between items-center p-2">
			<Sidebar.Trigger />
			<ModeToggle />
		</header>
		<main class="p-2 flex-1 flex flex-col">
			{@render children()}
		</main>
	</Sidebar.Inset>
</Sidebar.Provider>
