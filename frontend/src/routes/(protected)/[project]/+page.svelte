<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Grid2x2Plus, Search, ExternalLink } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { ChevronDown } from '@lucide/svelte';
	import { useGetAllServicesQuery } from '@/features/services';
	import { AppDeletion } from '@/components/conformation';
	import { DotmSquare } from '@/components/loader';
	import InstancePRPreviewDropdown from '@/components/InstancePRPreviewDropdown.svelte';
	import { X } from '@lucide/svelte';
	import type { PRInfo } from '@/features/services';
	import { PsqlService } from '@/components/services/predefined';
	import { InlinePanel } from '@/components/ui/inline-panel';
	import Icon from '@iconify/svelte';
	import { timeAgo } from '@/utils/time';

	let searchQuery = $state('');
	let selectedPR = $state<{ serviceName: string; pr: PRInfo } | null>(null);
	let selectedServiceId = $state('');
	let selectedServiceType = $state('');
	let drawerOpen = $state(false);

	const { data } = $props();
	const servicesQuery = useGetAllServicesQuery();

	const getProjectName = () => data.projectName;

	const filteredServices = $derived.by(() => {
		if (!servicesQuery.data) return [];

		const keyword = searchQuery.trim().toLowerCase();
		if (keyword === '') return servicesQuery.data;

		return servicesQuery.data.filter((service) => service.name.toLowerCase().includes(keyword));
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

	function getServiceLabel(type: string) {
		return type === 'app' ? 'App' : 'PostgreSQL';
	}

	function getServiceIcon(type: string) {
		return type === 'app' ? 'akar-icons:github-fill' : 'logos:postgresql';
	}

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
			drawerOpen = true;
		}
	}
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

<section class="relative flex-1 overflow-hidden p-4">
	{#if servicesQuery.isPending}
		<div class="size-full flex items-center justify-center">
			<DotmSquare size={65} dotSize={8} />
		</div>
	{:else if servicesQuery.isError}
		<p class="text-destructive">Failed to load services</p>
	{:else if filteredServices.length > 0}
		<div class="grid gap-3" style="grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));">
			{#each filteredServices as service (service.id)}
				{@const isApp = service.type === 'app'}
				<button
					class="group flex flex-col gap-3 rounded-lg border bg-card p-4 text-left transition-shadow hover:shadow-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
					onclick={() => handleServiceClick(service)}
				>
					<div class="flex items-start gap-3">
						<Icon icon={getServiceIcon(service.type)} class="mt-0.5 size-6 shrink-0" />
						<div class="min-w-0 flex-1">
							<p class="truncate font-semibold text-sm">{service.name}</p>
							<p class="text-xs text-muted-foreground">{timeAgo(service.created_at)}</p>
						</div>
						<div class="flex items-center gap-2 shrink-0">
							<span
								class="rounded-md bg-secondary px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-secondary-foreground"
							>
								{getServiceLabel(service.type)}
							</span>
							{#if isApp}
								<AppDeletion serviceId={service.id} name={service.name} />
							{/if}
						</div>
					</div>

					<div class="flex flex-col gap-2">
						{#if isApp}
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
								<Icon icon="lucide:git-branch" class="size-3 shrink-0" />
								<span class="truncate font-mono">{service.branch_name}</span>
							</div>
							<a
								href={service.gh_repo_url}
								target="_blank"
								rel="noopener noreferrer"
								onclick={(e) => e.stopPropagation()}
								class="inline-flex items-center gap-1.5 self-start rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted/80 hover:text-foreground no-underline"
							>
								<Icon icon="akar-icons:github-fill" class="size-3" />
								<span class="max-w-[160px] truncate">{service.gh_repo_name}</span>
								<ExternalLink class="size-3 shrink-0" />
							</a>
						{:else}
							<div class="flex items-center gap-2">
								{#if service.status === 'running'}
									<div class="flex items-center gap-1.5">
										<span class="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></span>
										<span class="text-xs font-medium text-emerald-600 dark:text-emerald-400"
											>Running</span
										>
									</div>
								{:else if service.status === 'paused'}
									<div class="flex items-center gap-1.5">
										<span class="h-2 w-2 rounded-full bg-amber-500"></span>
										<span class="text-xs font-medium text-amber-600 dark:text-amber-400"
											>Paused</span
										>
									</div>
								{:else}
									<span class="text-xs text-muted-foreground capitalize">{service.status}</span>
								{/if}
							</div>
						{/if}
					</div>
				</button>
			{/each}
		</div>
	{:else}
		<div class="size-full flex flex-col items-center justify-center gap-2">
			<Grid2x2Plus class="text-primary" />
			<h1>New Project</h1>
			<p class="text-muted-foreground text-sm">deploy your project</p>
		</div>
	{/if}

	<InlinePanel
		bind:open={drawerOpen}
		class="border-l bg-background shadow border border-r-0 w-2/3"
		showOverlay={true}
	>
		<div class="min-h-0 flex-1 overflow-y-auto">
			{#if selectedServiceType === 'psql'}
				<PsqlService serviceID={selectedServiceId} {drawerOpen} />
			{:else}
				<p class="p-4 text-muted-foreground">Service details coming soon</p>
			{/if}
		</div>
	</InlinePanel>
</section>
