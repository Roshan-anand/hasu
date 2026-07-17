<script lang="ts">
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { queryClient } from '@/query';
	import type { Component as SvelteComponentType } from 'svelte';

	let { component: Component }: { component: SvelteComponentType } = $props();
</script>

<QueryClientProvider client={queryClient}>
	{#if Component}
		<Component />
	{/if}
</QueryClientProvider>
