<script lang="ts">
	import favicon from '@/assets/favicon.svg';
	import { ModeWatcher } from 'mode-watcher';
	import '../app.css';
	import { Toaster } from '@/components/ui/sonner/index';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { queryClient } from '@/query';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<Toaster />
<ModeWatcher />
<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>
