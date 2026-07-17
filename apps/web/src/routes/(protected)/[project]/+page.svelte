<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Grid2x2Plus, Search, Network, Grid2x2, RotateCcw } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { ChevronDown } from '@lucide/svelte';
	import { useGetAllServicesQuery } from '@/features/services';
	import { DotmSquare } from '@/components/loader';
	import { PsqlService } from '@/components/services/predefined';
	import { InlinePanel } from '@/components/ui/inline-panel';
	import AppServiceCard from './AppServiceCard.svelte';
	import PsqlServiceCard from './PsqlServiceCard.svelte';
	import ProjectSettings from '@/components/settings/ProjectSettings.svelte';
	import { getInstanceState } from '@/features/instance/store.svelte.js';
	import GraphView from './GraphView.svelte';
	import { getBaseState } from '@/features/base';

	const base = getBaseState();

	let searchQuery = $state('');
	let selectedServiceId = $state('');
	let selectedServiceType = $state('');
	let viewMode = $state<'list' | 'graph'>('list');
	let graphViewRef = $state<GraphView | null>(null);

	const { data } = $props();
	const servicesQuery = useGetAllServicesQuery();
	const instance = getInstanceState();

	const getProjectName = () => data.projectName;

	// instance status derived from API message — cooking / deleting / ready
	const instanceStatus = $derived.by(() => {
		if (servicesQuery.isPending) return 'loading';
		if (!servicesQuery.data) return 'loading';
		return servicesQuery.data.message || 'ready';
	});

	const services = $derived(servicesQuery.data?.data ?? []);

	const filteredServices = $derived.by(() => {
		if (services.length === 0) return [];

		const keyword = searchQuery.trim().toLowerCase();
		if (keyword === '') return services;

		return services.filter((service) => service.name.toLowerCase().includes(keyword));
	});

	const createOptions = [
		{
			name: 'Application',
			link: resolve('/(protected)/[project]/new/app', {
				project: getProjectName()
			})
		},
		{
			name: 'DB',
			link: resolve('/(protected)/[project]/new/db', {
				project: getProjectName()
			})
		}
	];

	function handleServiceClick(service: { type: string; name: string; id: string }) {
		if (service.type === 'app') {
			goto(
				resolve('/(protected)/[project]/[service]', {
					service: service.name,
					project: getProjectName()
				})
			);
		} else {
			selectedServiceId = service.id;
			selectedServiceType = service.type;
			base.setPanelDrawerState(true);
		}
	}
</script>

<nav class="flex gap-4">
	<div class="flex-1 flex relative">
		<Input
			id="service-search"
			placeholder="Search for services"
			class="p-2"
			bind:value={searchQuery}
		/>
		<Label class="absolute top-0 right-0 m-1 opacity-75" for="service-search"><Search /></Label>
	</div>
	<ProjectSettings name={getProjectName()} projectId={instance.projectID} />
	{#if viewMode === 'graph'}
		<Button
			variant="ghost"
			size="icon"
			onclick={() => graphViewRef?.resetView()}
			title="Reset graph view"
		>
			<RotateCcw class="size-4" />
		</Button>
	{/if}
	<div class="flex items-center gap-1 rounded-md border bg-card p-0.5">
		<Button
			variant={viewMode === 'list' ? 'default' : 'ghost'}
			size="icon"
			class="size-8"
			onclick={() => (viewMode = 'list')}
			title="List view"
		>
			<Grid2x2 class="size-4" />
		</Button>
		<Button
			variant={viewMode === 'graph' ? 'default' : 'ghost'}
			size="icon"
			class="size-8"
			onclick={() => (viewMode = 'graph')}
			title="Graph view"
		>
			<Network class="size-4" />
		</Button>
	</div>
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button {...props}>
					<span>create</span>
					<ChevronDown class="size-4" />
				</Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end" class="w-40">
			{#each createOptions as option, i (option.name || i)}
				<!-- eslint-disable svelte/no-navigation-without-resolve -->
				<DropdownMenu.Item onSelect={() => goto(option.link)}>
					{option.name}
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
</nav>

<section class="relative flex-1 overflow-hidden p-4">
	{#if instanceStatus === 'loading'}
		<div class="size-full flex items-center justify-center">
			<DotmSquare size={65} dotSize={8} />
		</div>
	{:else if servicesQuery.isError}
		<p class="text-destructive">Failed to load services</p>
	{:else if instanceStatus === 'cooking'}
		<div class="size-full flex flex-col items-center justify-center gap-2">
			<DotmSquare size={65} dotSize={8} />
			<p class="text-muted-foreground text-sm">cooking the instance</p>
		</div>
	{:else if instanceStatus === 'deleting'}
		<div class="size-full flex flex-col items-center justify-center gap-2">
			<DotmSquare size={65} dotSize={8} />
			<p class="text-muted-foreground text-sm">deleting the instance</p>
		</div>
	{:else if filteredServices.length > 0}
		{#if viewMode === 'graph'}
			<GraphView
				bind:this={graphViewRef}
				services={filteredServices}
				onNodeClick={handleServiceClick}
			/>
		{:else}
			<div class="grid gap-3" style="grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));">
				{#each filteredServices as service, i (service.id || i)}
					{#if service.type === 'app'}
						<AppServiceCard {service} onclick={() => handleServiceClick(service)} />
					{:else}
						<PsqlServiceCard {service} onclick={() => handleServiceClick(service)} />
					{/if}
				{/each}
			</div>
		{/if}
	{:else}
		<div class="size-full flex flex-col items-center justify-center gap-2">
			<Grid2x2Plus class="text-primary" />
			<h1>New Project</h1>
			<p class="text-muted-foreground text-sm">deploy your project</p>
		</div>
	{/if}

	<InlinePanel
		bind:open={base.inlinePanelDrawer}
		class="border-l bg-background shadow border border-r-0 w-2/3"
		showOverlay={true}
	>
		<div class="min-h-0 flex-1 overflow-y-auto">
			{#if selectedServiceType === 'psql'}
				<PsqlService serviceID={selectedServiceId} />
			{:else}
				<p class="p-4 text-muted-foreground">Service details coming soon</p>
			{/if}
		</div>
	</InlinePanel>
</section>
