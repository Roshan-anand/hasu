<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Skeleton } from '@/components/ui/skeleton';
	import { useDeleteDeploymentMutation } from '@/features/deployments';
	import { useServiceDeploymentsQuery } from '@/features/deployments';
	import DeployementLogs from './deployement_logs.svelte';

	let { serviceID }: { serviceID: string } = $props();

	// so the list updates immediately without waiting for a refetch.
	const deploymentsQuery = useServiceDeploymentsQuery(() => serviceID);

	const deleteDeploymentMutation = useDeleteDeploymentMutation(() => serviceID);

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

			{#each deploymentsQuery.data as { id, commit_msg, status, created_at, is_current } (id)}
				<div
					class="grid grid-cols-[1.5fr_1fr_1fr_auto] items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0"
				>
					<p class="truncate">{commit_msg}</p>
					<p class="capitalize">{status}</p>
					<p>{new Date(created_at).toLocaleString()}</p>

					<div class="flex items-center justify-end gap-2">
						<!-- TODO : impliment deployment name  -->
						<DeployementLogs deploymentId={id} deploymentName={id} />

						{#if !is_current}
							<Button
								variant="destructive"
								size="sm"
								disabled={deleteDeploymentMutation.isPending}
								onclick={() => deleteDeployment(id)}
							>
								Delete
							</Button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>
