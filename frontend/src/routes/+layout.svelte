<script lang="ts">
	import favicon from '@/assets/favicon.svg';
	import { ModeWatcher } from 'mode-watcher';
	import '../app.css';
	import { Toaster } from '@/components/ui/sonner/index';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import { queryClient } from '@/query';
	import { setUserState } from '@/features/global/store.svelte';

	let { children } = $props();

	setUserState();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<Toaster />
<ModeWatcher />
<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>
