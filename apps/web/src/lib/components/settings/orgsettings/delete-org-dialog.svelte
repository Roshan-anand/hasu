<script lang="ts">
	import * as Dialog from '@/components/ui/dialog';
	import { Button } from '@/components/ui/button';
	import { Skeleton } from '@/components/ui/skeleton';
	import { HardDrive } from '@lucide/svelte';
	import {
		useDeleteOrgMutation,
		useGetOrgProjectsQuery,
		useGetOrgVolumesQuery,
		getOrgState,
		useSwitchOrgMutation
	} from '@/features/base';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import TransferVolumeDialog from './transfer-volume-dialog.svelte';
	import TransferProjectDialog from './transfer-project-dialog.svelte';

	type Org = {
		id: string;
		name: string;
	};

	type Props = {
		open?: boolean;
		orgId?: string;
		orgName?: string;
		orgs?: Org[];
	};

	let { open = $bindable(false), orgId = '', orgName = '', orgs = [] }: Props = $props();

	const currentOrg = getOrgState();
	const deleteOrgMutation = useDeleteOrgMutation();
	const switchOrgMutation = useSwitchOrgMutation();

	const orgProjectsQuery = useGetOrgProjectsQuery(() => (open ? orgId : ''));
	const orgVolumesQuery = useGetOrgVolumesQuery(() => (open ? orgId : ''));

	// Single nullable objects — the transfer dialogs manage their own open/close state
	let volumeToTransfer = $state<{ id: string; name: string } | null>(null);
	let projectToTransfer = $state<{ id: string; name: string } | null>(null);

	function confirmDelete() {
		if (deleteOrgMutation.isPending) return;
		const wasCurrentOrg = currentOrg.id === orgId;
		deleteOrgMutation.mutate(
			{ org_id: orgId },
			{
				onSuccess: () => {
					open = false;
					if (wasCurrentOrg) {
						const remainingOrgs = orgs.filter((o) => o.id !== orgId);
						if (remainingOrgs.length > 0) {
							switchOrgMutation.mutate({ org_id: remainingOrgs[0].id });
						}
						goto(resolve('/'));
					}
				}
			}
		);
	}

	const projects = $derived(orgProjectsQuery.data ?? []);
	const volumes = $derived(orgVolumesQuery.data ?? []);
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg max-h-[80vh] overflow-y-auto"
		>
			<Dialog.Title class="text-lg font-semibold text-destructive">Delete Organization</Dialog.Title
			>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Are you sure you want to delete <strong>{orgName}</strong>? This action cannot be undone.
			</Dialog.Description>

			<!-- Projects warning -->
			{#if orgProjectsQuery.isPending}
				<div class="mt-4">
					<Skeleton class="h-16 w-full" />
				</div>
			{:else if projects.length > 0}
				<div class="mt-4 space-y-2">
					<p class="text-sm font-medium text-destructive">
						The following projects and their services will be permanently removed:
					</p>
					<ul class="space-y-1">
						{#each projects as project (project.id)}
							<li
								class="flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2 text-sm"
							>
								<span>{project.name}</span>
								<div class="flex items-center gap-2">
									<Button
										variant="outline"
										size="sm"
										onclick={() => (projectToTransfer = { id: project.id, name: project.name })}
									>
										Transfer
									</Button>
								</div>
							</li>
						{/each}
					</ul>
				</div>
			{:else}
				<p class="mt-4 text-sm text-muted-foreground">
					No projects in this organization. It can be safely deleted.
				</p>
			{/if}

			<!-- Volumes warning -->
			{#if orgVolumesQuery.isPending}
				<div class="mt-4">
					<Skeleton class="h-16 w-full" />
				</div>
			{:else if volumes.length > 0}
				<div class="mt-4 space-y-2">
					<p class="text-sm font-medium text-muted-foreground">
						The following orphan storage volumes will also be removed:
					</p>
					<ul class="space-y-1">
						{#each volumes as vol (vol.id)}
							<li
								class="flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2 text-sm"
							>
								<div class="flex items-center gap-2 min-w-0">
									<HardDrive class="size-4 shrink-0 text-muted-foreground" />
									<span class="truncate">{vol.volume}</span>
									<span
										class="shrink-0 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground uppercase"
										>{vol.type}</span
									>
								</div>
								<div class="flex items-center gap-2 shrink-0">
									<Button
										variant="outline"
										size="sm"
										onclick={() => (volumeToTransfer = { id: vol.id, name: vol.volume })}
									>
										Transfer
									</Button>
								</div>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			<div class="flex justify-end gap-2 mt-6 pt-2 border-t">
				<Button
					variant="outline"
					type="button"
					onclick={() => (open = false)}
					disabled={deleteOrgMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					variant="destructive"
					onclick={confirmDelete}
					disabled={deleteOrgMutation.isPending || orgProjectsQuery.isPending}
				>
					{deleteOrgMutation.isPending ? 'Deleting...' : 'Delete Organization'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<!-- Transfer Volume Dialog — manages its own open/close via the `volume` object -->
<TransferVolumeDialog volume={volumeToTransfer} {orgs} sourceOrgId={orgId} />

<!-- Transfer Project Dialog — manages its own open/close via the `project` object -->
<TransferProjectDialog project={projectToTransfer} {orgs} sourceOrgId={orgId} />
