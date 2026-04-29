<script lang="ts">
	import CreateBtn from '@/components/CreateBtn.svelte';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Search } from '@lucide/svelte';
	import CreateServiceDialog from './CreateServiceDialog.svelte';
	import { getUserState } from '@/features/global/store.svelte';
	import { setServiceState } from '@/features/services/store.svelte';

	let { children } = $props();
	const { currentOrg } = getUserState();

	// shared UI state for this page (search, dialog open/close)
	const serviceState = setServiceState();
</script>

<nav class="flex gap-2">
	<div class="flex-1 flex relative">
		<Input
			id="service-search"
			placeholder="Search for services"
			class="p-2"
			bind:value={serviceState.searchQuery}
		/>
		<Label class="absolute top-0 right-0 m-1 opacity-75" for="service-search"><Search /></Label>
	</div>
	<CreateBtn onclick={serviceState.openCreateDialog} disabled={currentOrg.id === ''} />
</nav>

<CreateServiceDialog />

{@render children()}
