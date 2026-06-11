<script lang="ts">
	import * as Dialog from '@/components/ui/dialog';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import * as Card from '@/components/ui/card';
	import { Check, Plus, Pencil, Trash2, ArrowRight, Building2 } from '@lucide/svelte';
	import {
		getOrgState,
		useGetAllOrgsQuery,
		useSwitchOrgMutation,
		useCreateOrgMutation,
		useRenameOrgMutation,
		useDeleteOrgMutation,
		useTransferProjectMutation,
		useGetOrgProjectsQuery
	} from '@/features/base';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';

	const currentOrg = getOrgState();

	const getAllOrgsQuery = useGetAllOrgsQuery();
	const switchOrgMutation = useSwitchOrgMutation();
	const createOrgMutation = useCreateOrgMutation();
	const renameOrgMutation = useRenameOrgMutation();
	const deleteOrgMutation = useDeleteOrgMutation();
	const transferProjectMutation = useTransferProjectMutation();

	// Create org dialog
	let createDialogOpen = $state(false);
	let createOrgName = $state('');

	// Rename org dialog
	let renameDialogOpen = $state(false);
	let renameOrgId = $state('');
	let renameOrgName = $state('');

	// Delete org dialog
	let deleteDialogOpen = $state(false);
	let deleteOrgId = $state('');
	let deleteOrgName = $state('');

	// Transfer project dialog (inside delete flow)
	let transferDialogOpen = $state(false);
	let transferProjectId = $state('');
	let transferProjectName = $state('');
	let transferTargetOrgId = $state('');

	const orgs = $derived(getAllOrgsQuery.data ?? []);
	const deleteOrgProjectsQuery = useGetOrgProjectsQuery(() =>
		deleteDialogOpen ? deleteOrgId : ''
	);

	const targetOrgs = $derived(orgs.filter((o) => o.id !== deleteOrgId));

	function switchOrg(orgId: string) {
		if (!orgId || orgId === currentOrg.id || switchOrgMutation.isPending) return;
		switchOrgMutation.mutate({ org_id: orgId });
	}

	function openCreateDialog() {
		createOrgName = '';
		createDialogOpen = true;
	}

	function createOrg() {
		if (createOrgName.trim().length < 3 || createOrgMutation.isPending) return;
		createOrgMutation.mutate(
			{ name: createOrgName.trim() },
			{
				onSuccess: () => {
					createOrgName = '';
					createDialogOpen = false;
				}
			}
		);
	}

	function openRenameDialog(orgId: string, orgName: string) {
		renameOrgId = orgId;
		renameOrgName = orgName;
		renameDialogOpen = true;
	}

	function renameOrg() {
		if (renameOrgName.trim().length < 3 || renameOrgMutation.isPending) return;
		renameOrgMutation.mutate(
			{ org_id: renameOrgId, name: renameOrgName.trim() },
			{
				onSuccess: () => {
					renameDialogOpen = false;
				}
			}
		);
	}

	function openDeleteDialog(orgId: string, orgName: string) {
		deleteOrgId = orgId;
		deleteOrgName = orgName;
		deleteDialogOpen = true;
	}

	function handleTransferProject(projectId: string, projectName: string) {
		transferProjectId = projectId;
		transferProjectName = projectName;
		transferTargetOrgId = '';
		transferDialogOpen = true;
	}

	function confirmTransfer() {
		if (!transferTargetOrgId || transferProjectMutation.isPending) return;
		transferProjectMutation.mutate(
			{ project_id: transferProjectId, target_org_id: transferTargetOrgId },
			{
				onSuccess: () => {
					transferDialogOpen = false;
					toast.success(`Project "${transferProjectName}" transferred`);
					// Refresh the project list for delete warning
				}
			}
		);
	}

	function confirmDeleteOrg() {
		if (deleteOrgMutation.isPending) return;
		const wasCurrentOrg = currentOrg.id === deleteOrgId;
		deleteOrgMutation.mutate(
			{ org_id: deleteOrgId },
			{
				onSuccess: () => {
					deleteDialogOpen = false;
					// If we just deleted the current org, switch to first available and redirect
					if (wasCurrentOrg) {
						if (orgs.length > 1) {
							const nextOrg = orgs.find((o) => o.id !== deleteOrgId);
							if (nextOrg) {
								switchOrgMutation.mutate({ org_id: nextOrg.id });
							}
						}
						goto(resolve('/'));
					}
				}
			}
		);
	}

	const isOrgsLoading = $derived(getAllOrgsQuery.isPending);
</script>

<section class="space-y-6">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-medium">Organizations</h2>
		<Button size="sm" onclick={openCreateDialog} disabled={createOrgMutation.isPending}>
			<Plus class="size-4" />
			<span class="ml-1">Create</span>
		</Button>
	</div>

	{#if isOrgsLoading}
		<div class="space-y-3">
			<Skeleton class="h-20 w-full" />
			<Skeleton class="h-20 w-full" />
		</div>
	{:else if orgs.length === 0}
		<p class="text-muted-foreground text-sm">No organizations yet. Create one to get started.</p>
	{:else}
		<div class="space-y-3">
			{#each orgs as org (org.id)}
				<Card.Root class="p-4">
					<div class="flex items-center justify-between gap-4">
						<div class="flex items-center gap-3 min-w-0">
							<div
								class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10"
							>
								<Building2 class="size-5 text-primary" />
							</div>
							<div class="min-w-0">
								<p class="truncate font-medium">{org.name}</p>
								<p class="text-xs text-muted-foreground">
									{#if org.id === currentOrg.id}
										<span class="text-primary font-medium">Current organization</span>
									{:else}
										Click to switch
									{/if}
								</p>
							</div>
						</div>

						<div class="flex items-center gap-1 shrink-0">
							<Button
								variant="ghost"
								size="icon"
								title="Rename"
								onclick={() => openRenameDialog(org.id, org.name)}
							>
								<Pencil class="size-4" />
							</Button>

							<Button
								variant="ghost"
								size="icon"
								title="Switch to this organization"
								disabled={org.id === currentOrg.id || switchOrgMutation.isPending}
								onclick={() => switchOrg(org.id)}
							>
								<ArrowRight class="size-4" />
							</Button>

							<Button
								variant="ghost"
								size="icon"
								title="Delete organization"
								disabled={org.id === currentOrg.id}
								class="hover:text-destructive"
								onclick={() => openDeleteDialog(org.id, org.name)}
							>
								<Trash2 class="size-4" />
							</Button>
						</div>
					</div>
				</Card.Root>
			{/each}
		</div>
	{/if}
</section>

<Dialog.Root bind:open={createDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Create Organization</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Add a name for your organization.
			</Dialog.Description>

			<form
				class="mt-4 space-y-4"
				onsubmit={(e) => {
					e.preventDefault();
					createOrg();
				}}
			>
				<div class="space-y-1.5">
					<Label for="settings-create-org-name">Name</Label>
					<Input
						id="settings-create-org-name"
						placeholder="Organization name"
						bind:value={createOrgName}
						required
						minlength={3}
						disabled={createOrgMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={() => (createDialogOpen = false)}
						disabled={createOrgMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={createOrgMutation.isPending}>
						{createOrgMutation.isPending ? 'Creating...' : 'Create'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<!-- Rename Org Dialog -->
<Dialog.Root bind:open={renameDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Rename Organization</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Enter a new name for this organization.
			</Dialog.Description>

			<form
				class="mt-4 space-y-4"
				onsubmit={(e) => {
					e.preventDefault();
					renameOrg();
				}}
			>
				<div class="space-y-1.5">
					<Label for="settings-rename-org-name">Name</Label>
					<Input
						id="settings-rename-org-name"
						placeholder="New organization name"
						bind:value={renameOrgName}
						required
						minlength={3}
						disabled={renameOrgMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={() => (renameDialogOpen = false)}
						disabled={renameOrgMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={renameOrgMutation.isPending}>
						{renameOrgMutation.isPending ? 'Saving...' : 'Save'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<!-- Delete Org Dialog -->
<Dialog.Root bind:open={deleteDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg max-h-[80vh] overflow-y-auto"
		>
			<Dialog.Title class="text-lg font-semibold text-destructive">Delete Organization</Dialog.Title
			>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Are you sure you want to delete <strong>{deleteOrgName}</strong>? This action cannot be
				undone.
			</Dialog.Description>

			<!-- Projects warning -->
			{#if deleteOrgProjectsQuery.isPending}
				<div class="mt-4">
					<Skeleton class="h-16 w-full" />
				</div>
			{:else if deleteOrgProjectsQuery.data && deleteOrgProjectsQuery.data.length > 0}
				<div class="mt-4 space-y-2">
					<p class="text-sm font-medium text-destructive">
						The following projects and their services will be permanently removed:
					</p>
					<ul class="space-y-1">
						{#each deleteOrgProjectsQuery.data as project (project.id)}
							<li
								class="flex items-center justify-between rounded-md border bg-muted/30 px-3 py-2 text-sm"
							>
								<span>{project.name}</span>
								<div class="flex items-center gap-2">
									<Button
										variant="outline"
										size="sm"
										onclick={() => handleTransferProject(project.id, project.name)}
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

			<div class="flex justify-end gap-2 mt-6 pt-2 border-t">
				<Button
					variant="outline"
					type="button"
					onclick={() => (deleteDialogOpen = false)}
					disabled={deleteOrgMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					variant="destructive"
					onclick={confirmDeleteOrg}
					disabled={deleteOrgMutation.isPending || deleteOrgProjectsQuery.isPending}
				>
					{deleteOrgMutation.isPending ? 'Deleting...' : 'Delete Organization'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<!-- Transfer Project Dialog (inside delete flow) -->
<Dialog.Root bind:open={transferDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-50 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Transfer Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Move <strong>{transferProjectName}</strong> to another organization instead of deleting it.
			</Dialog.Description>

			<div class="mt-4 space-y-3">
				<Label for="transfer-target-org">Target Organization</Label>
				{#if targetOrgs.length === 0}
					<p class="text-sm text-muted-foreground">
						No other organizations available. Create one first.
					</p>
				{:else}
					<div class="space-y-2">
						{#each targetOrgs as org (org.id)}
							<button
								type="button"
								onclick={() => (transferTargetOrgId = org.id)}
								class="flex w-full items-center gap-3 rounded-lg border px-3 py-2 text-left text-sm transition-colors hover:bg-accent
									{transferTargetOrgId === org.id ? 'border-primary ring-2 ring-primary/20' : 'border-border'}"
							>
								<div class="flex size-8 items-center justify-center rounded-md bg-primary/10">
									<Building2 class="size-4 text-primary" />
								</div>
								<span class="flex-1 font-medium">{org.name}</span>
								{#if transferTargetOrgId === org.id}
									<Check class="size-4 text-primary" />
								{/if}
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<div class="flex justify-end gap-2 mt-6 pt-2 border-t">
				<Button
					variant="outline"
					type="button"
					onclick={() => (transferDialogOpen = false)}
					disabled={transferProjectMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					onclick={confirmTransfer}
					disabled={!transferTargetOrgId || transferProjectMutation.isPending}
				>
					{transferProjectMutation.isPending ? 'Transferring...' : 'Transfer'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
