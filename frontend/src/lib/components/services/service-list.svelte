<script lang="ts">
	import { api, axiosErr } from '@/axios';
	import CreateBtn from '@/components/CreateBtn.svelte';
	import { Button } from '@/components/ui/button';
	import { Skeleton } from '@/components/ui/skeleton';
	import { queryClient } from '@/query';
	import { createMutation, createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import { Trash2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { getServiceState } from '@/features/services/store.svelte';

	type ServiceType = 'psql' | 'app';
	type ScopeType = 'project' | 'org';

	type ServiceRow = {
		id: string;
		type: ServiceType;
		name: string;
		description: string;
		created_at: string;
	};

	type ServiceListResponse = {
		services: ServiceRow[];
	};

	type DeleteServicePayload = {
		service_id: string;
		type: ServiceType;
	};

	type DeleteResponse = {
		message: string;
	};

	let { scopeId, scopeType } = $props<{
		scopeId: string;
		scopeType: ScopeType;
	}>();

	let deletingServiceId = $state('');
	const serviceState = getServiceState();

	const getServiceListQueryKey = () => ['services', scopeType, scopeId];

	// Shared service-list query/mutation keeps project-level and org-level pages in one UI component.
	const servicesQuery = createQuery(() => ({
		queryKey: getServiceListQueryKey(),
		queryFn: () => {
			const params = scopeType === 'project' ? { project_id: scopeId } : {};
			const url = scopeType === 'project' ? '/service/project' : '/service/org';

			return api.get<ServiceListResponse>(url, { params }).then((res) => res.data.services);
		},
		enabled: scopeId !== ''
	}));

	const deleteServiceMutation = createMutation(() => ({
		mutationFn: ({ service_id, type }: DeleteServicePayload) => {
			const url = type === 'psql' ? '/service/psql' : '/service/app';
			return api.delete<DeleteResponse>(url, { data: { service_id } }).then((res) => res.data);
		},
		onMutate: ({ service_id }) => {
			deletingServiceId = service_id;
		},
		onSuccess: (res, payload) => {
			queryClient.setQueryData(getServiceListQueryKey(), (cachedRows: ServiceRow[] | undefined) => {
				if (!cachedRows) return [];
				return cachedRows.filter((row) => row.id !== payload.service_id);
			});
			toast.success(res.message || 'Service deleted successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to delete service'),
		onSettled: () => {
			deletingServiceId = '';
		}
	}));

	const deleteService = (serviceId: string, type: ServiceType) => {
		if (deleteServiceMutation.isPending) return;
		deleteServiceMutation.mutate({ service_id: serviceId, type });
	};

	const filteredServices = $derived.by(() => {
		if (!servicesQuery.data) return [];

		const keyword = serviceState.searchQuery.trim().toLowerCase();
		if (keyword === '') return servicesQuery.data;

		return servicesQuery.data.filter((service) => service.name.toLowerCase().includes(keyword));
	});

	const tempItem = Array.from({ length: 6 });
</script>

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
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each filteredServices as service (service.id)}
				<div
					class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 hover:shadow-md transition-shadow cursor-pointer relative"
				>
					<a
						href={resolve('/(core)/service/[service_type]/[service_id]', {
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
							{#if deleteServiceMutation.isPending && deletingServiceId === service.id}
								Deleting...
							{:else}
								<Trash2 />
								Delete
							{/if}
						</Button>
					</div>
					<p class="text-muted-foreground text-sm line-clamp-2">
						{service.description || 'No description'}
					</p>
				</div>
			{/each}
		</div>
	{:else}
		<h3 class="text-muted-foreground size-full flex flex-col items-center justify-center gap-2">
			<span>No services found</span>
			<CreateBtn onclick={serviceState.openCreateDialog} />
		</h3>
	{/if}
</section>
