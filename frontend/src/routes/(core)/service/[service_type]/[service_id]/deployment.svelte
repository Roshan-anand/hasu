<script lang="ts">
	import { api, axiosErr } from '@/axios';
	import { Button } from '@/components/ui/button';
	import { Skeleton } from '@/components/ui/skeleton';
	import { queryClient } from '@/query';
	import type { ServiceDeployment } from '@/types.js';
	import { createMutation, createQuery } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import DeploymentLogs from './deployement_logs.svelte';

	type DeleteDeploymentPayload = {
		deployment_id: string;
	};

	type DeleteDeploymentResponse = {
		message: string;
	};

	let { serviceId }: { serviceId: string } = $props();
	let deletingDeploymentId = $state('');

	const deploymentsQueryKey = $derived(['service-deployments', serviceId]);

	// AI summary: Query deployments by current service id, and update query cache on delete
	// so the list updates immediately without waiting for a refetch.
	const deploymentsQuery = createQuery(() => ({
		queryKey: deploymentsQueryKey,
		queryFn: async () => {
			return api
				.get<ServiceDeployment[]>('/service/deployment', {
					params: { service_id: serviceId }
				})
				.then((res) => res.data);
		},
		enabled: serviceId !== ''
	}));

	const deleteDeploymentMutation = createMutation(() => ({
		mutationFn: async ({ deployment_id }: DeleteDeploymentPayload) => {
			return api
				.delete<DeleteDeploymentResponse>('/service/deployment', {
					data: { deployment_id }
				})
				.then((res) => res.data);
		},
		onMutate: ({ deployment_id }) => {
			deletingDeploymentId = deployment_id;
		},
		onSuccess: (res, payload) => {
			queryClient.setQueryData(
				deploymentsQueryKey,
				(cachedRows: ServiceDeployment[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== payload.deployment_id);
				}
			);
			toast.success(res.message || 'Deployment deleted successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to delete deployment'),
		onSettled: () => {
			deletingDeploymentId = '';
		}
	}));

	function deleteDeployment(deploymentId: string) {
		if (deleteDeploymentMutation.isPending) return;
		deleteDeploymentMutation.mutate({ deployment_id: deploymentId });
	}
</script>

<section class="rounded-lg border bg-card p-3 text-card-foreground shadow-sm">
	<h3 class="text-base font-semibold">Deployments</h3>

	{#if deploymentsQuery.isPending}
		<div class="mt-3 flex flex-col gap-2">
			{#each Array.from({ length: 4 }) as _, i (i)}
				<div class="rounded-md border p-3">
					<Skeleton class="h-5 w-1/3" />
				</div>
			{/each}
		</div>
	{:else if deploymentsQuery.isError}
		<p class="mt-3 text-sm text-red-500">Failed to load deployments</p>
	{:else if !deploymentsQuery.data || deploymentsQuery.data.length === 0}
		<p class="mt-3 text-sm text-muted-foreground">No deployments found for this service</p>
	{:else}
		<div class="mt-3 overflow-hidden rounded-md border">
			<div
				class="grid grid-cols-[1.5fr_1fr_1fr_auto] gap-2 border-b bg-muted/40 px-3 py-2 text-xs font-medium text-muted-foreground"
			>
				<span>Name</span>
				<span>Status</span>
				<span>Created</span>
				<span class="text-right">Actions</span>
			</div>

			{#each deploymentsQuery.data as deployment (deployment.id)}
				<div
					class="grid grid-cols-[1.5fr_1fr_1fr_auto] items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0"
				>
					<p class="truncate">{deployment.name}</p>
					<p class="capitalize">{deployment.status}</p>
					<p>{new Date(deployment.created_at).toLocaleString()}</p>

					<div class="flex items-center justify-end gap-2">
						<DeploymentLogs deploymentId={deployment.id} deploymentName={deployment.name} />
						<Button
							variant="destructive"
							size="sm"
							disabled={deleteDeploymentMutation.isPending}
							onclick={() => deleteDeployment(deployment.id)}
						>
							{#if deleteDeploymentMutation.isPending && deletingDeploymentId === deployment.id}
								Deleting...
							{:else}
								Delete
							{/if}
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>
