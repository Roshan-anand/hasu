<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Grid2x2Plus, Search } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { ChevronDown } from '@lucide/svelte';
	import { useGetAllServicesQuery } from '@/features/services';
	import AppDeletion from '@/components/conformation/app-deletion.svelte';
	import DbDeletion from '@/components/conformation/db-deletion.svelte';
	import { DotmSquare } from '@/components/loader';
	import InstancePRPreviewDropdown from '@/components/InstancePRPreviewDropdown.svelte';
	import { X } from '@lucide/svelte';
	import type { PRInfo } from '@/features/services';

	let searchQuery = $state('');
	let selectedPR = $state<{ serviceName: string; pr: PRInfo } | null>(null);

	const { data } = $props();
	const servicesQuery = useGetAllServicesQuery();

	const getProjectName = () => data.projectName;

	// to filter services based on search input
	const filteredServices = $derived.by(() => {
		if (!servicesQuery.data) return [];

		const keyword = searchQuery.trim().toLowerCase();
		console.log('keyword :', keyword);
		console.log('dta a', servicesQuery.data);
		if (keyword === '') return servicesQuery.data;

		const data = servicesQuery.data.filter((service) =>
			service.name.toLowerCase().includes(keyword)
		);
		console.log('after filer :', data);
		return data;
	});

	$effect(() => {
		console.log('filter :', filteredServices);
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
</script>

{#if selectedPR}
	<div
		class="mb-3 p-2 bg-accent text-accent-foreground rounded-lg flex items-center justify-between border"
	>
		<div class="flex items-center gap-2">
			<span class="text-xs bg-primary text-primary-foreground px-2 py-0.5 rounded font-semibold"
				>PR Preview</span
			>
			<span class="font-medium text-sm"
				>{selectedPR.serviceName} - #{selectedPR.pr.number}: {selectedPR.pr.title}</span
			>
		</div>
		<Button variant="ghost" size="icon" class="h-6 w-6" onclick={() => (selectedPR = null)}>
			<X class="h-4 w-4" />
		</Button>
	</div>
{/if}

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
	<InstancePRPreviewDropdown onSelect={(serviceName, pr) => (selectedPR = { serviceName, pr })} />
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
			{#each createOptions as option (option.name)}
				<!-- eslint-disable svelte/no-navigation-without-resolve -->
				<DropdownMenu.Item onSelect={() => goto(option.link)}>
					{option.name}
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
</nav>
<section class="flex-1 p-2">
	{#if servicesQuery.isPending}
		<div class="size-full flex items-center justify-center">
			<DotmSquare size={65} dotSize={8} />
		</div>
	{:else if servicesQuery.isError}
		<p class="text-red-500">Failed to load services</p>
	{:else if filteredServices.length > 0}
		<!-- TODO : update UI to include new type of app service returned -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each filteredServices as service, i (i)}
				<div
					class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 hover:shadow-md transition-shadow cursor-pointer relative"
				>
					{#if service.type}
						<a
							href={resolve('/(protected)/[project]/[service_type]/[service]', {
								service_type: service.type,
								service: service.name,
								project: getProjectName()
							})}
							class="absolute z-10 size-full inset-0 text-transparent"
							title="open service"
						></a>
					{/if}
					<div class="flex items-start justify-between gap-2">
						<div>
							<h3 class="font-semibold text-lg">{service.name}</h3>
							<p class="text-xs uppercase text-muted-foreground">{service.type}</p>
						</div>
						{#if service.type === 'app'}
							<AppDeletion serviceId={service.id} name={service.name} />
						{:else if service.type === 'psql'}
							<DbDeletion serviceId={service.id} name={service.name} />
						{/if}
					</div>
					<div>{service.type}</div>
				</div>
			{/each}
		</div>
	{:else}
		<div class="size-full flex flex-col items-center justify-center gap-2">
			<Grid2x2Plus class="text-primary" />
			<h1>New Project</h1>
			<p class="text-muted-foreground text-sm">deploy your project</p>
		</div>
	{/if}
</section>
