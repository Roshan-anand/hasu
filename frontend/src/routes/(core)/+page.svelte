<script lang="ts">
	import CreateBtn from '@/components/CreateBtn.svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { getUserState } from '@/features/global/store.svelte';
	import { useDeleteServiceMutation } from '@/features/services/mutation.svelte';
	import { setServiceState } from '@/features/services/store.svelte';
	import type { ServiceType } from '@/types';
	import { Search, Trash2 } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { useGetServicesQuery } from '@/features/services/query.svelte';
	import CreateServiceDialog from '@/components/services/CreateServiceDialog.svelte';

	const { currentOrg } = getUserState();
	const serviceState = setServiceState();
	let searchQuery = $state('');

	const servicesQuery = useGetServicesQuery();
	const deleteServiceMutation = useDeleteServiceMutation();

	const deleteService = (serviceId: string, type: ServiceType) => {
		if (deleteServiceMutation.isPending) return;
		deleteServiceMutation.mutate({ service_id: serviceId, type });
	};

	// to filter services based on search input
	const filteredServices = $derived.by(() => {
		if (!servicesQuery.data) return [];

		const keyword = searchQuery.trim().toLowerCase();
		if (keyword === '') return servicesQuery.data;

		return servicesQuery.data.filter((service) => service.name.toLowerCase().includes(keyword));
	});

	const tempItem = Array.from({ length: 6 });
</script>

<nav class="flex gap-2">
	<div class="flex-1 flex relative">
		<Input
			id="service-search"
			placeholder="Search for services"
			class="p-2"
			bind:value={searchQuery}
		/>
		<Label class="absolute top-0 right-0 m-1 opacity-75" for="service-search"><Search /></Label>
	</div>
	<CreateBtn onclick={serviceState.openCreateDialog} disabled={currentOrg.id === ''} />
</nav>

<CreateServiceDialog />

<section class="flex-1 p-2">
	{#if servicesQuery.isPending}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each tempItem as _, i (i)}
				<div class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 space-y-3">
					<Skeleton class="h-6 w-3/4" />
					<Skeleton class="h-4 w-1/2" />
				</div>
			{/each}
		</div>
	{:else if servicesQuery.isError}
		<p class="text-red-500">Failed to load services</p>
	{:else if filteredServices.length > 0}
		<!-- TODO : update UI to include new type of app service returned -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each filteredServices as service (service.id)}
				<div
					class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 hover:shadow-md transition-shadow cursor-pointer relative"
				>
					<a
						href={resolve('/(core)/[service_type]/[service_id]', {
							service_type: service.type,
							service_id: service.id
						})}
						class="absolute z-10 size-full inset-0 text-transparent"
						title="open service"
					></a>
					<div class="flex items-start justify-between gap-2">
						<div>
							<h3 class="font-semibold text-lg">{service.name}</h3>
							<p class="text-xs uppercase text-muted-foreground">{service.type}</p>
						</div>
						<Button
							variant="destructive"
							size="sm"
							onclick={() => deleteService(service.id, service.type)}
							disabled={deleteServiceMutation.isPending}
							class="z-20"
						>
							{#if deleteServiceMutation.isPending}
								Deleting...
							{:else}
								<Trash2 />
								Delete
							{/if}
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<h3 class="text-muted-foreground size-full flex flex-col items-center justify-center gap-2">
			<span>No services found</span>
			<CreateBtn onclick={serviceState.openCreateDialog} disabled={currentOrg.id === ''} />
		</h3>
	{/if}
</section>
