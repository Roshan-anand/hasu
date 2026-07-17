<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import * as Card from '@/components/ui/card';
	import { Skeleton } from '@/components/ui/skeleton';
	import { Plus, Pencil, Trash2, ArrowRight, Building2, Settings2, Search } from '@lucide/svelte';
	import { getOrgState, useGetAllOrgsQuery, useSwitchOrgMutation } from '@/features/base';

	import CreateOrgDialog from './create-org-dialog.svelte';
	import RenameOrgDialog from './rename-org-dialog.svelte';
	import DeleteOrgDialog from './delete-org-dialog.svelte';

	const currentOrg = getOrgState();

	const getAllOrgsQuery = useGetAllOrgsQuery();
	const switchOrgMutation = useSwitchOrgMutation();

	// Create org dialog
	let createDialogOpen = $state(false);

	// Rename org dialog
	let renameDialogOpen = $state(false);
	let renameOrgId = $state('');
	let renameOrgName = $state('');

	// Delete org dialog
	let deleteDialogOpen = $state(false);
	let deleteOrgId = $state('');
	let deleteOrgName = $state('');

	let orgSearchQuery = $state('');

	const orgs = $derived(getAllOrgsQuery.data ?? []);
	const filteredOrgs = $derived(
		orgSearchQuery.trim() === ''
			? orgs
			: orgs.filter((o) => o.name.toLowerCase().includes(orgSearchQuery.trim().toLowerCase()))
	);

	function switchOrg(orgId: string) {
		if (!orgId || orgId === currentOrg.id || switchOrgMutation.isPending) return;
		switchOrgMutation.mutate({ org_id: orgId });
	}

	function openCreateDialog() {
		createDialogOpen = true;
	}

	function openRenameDialog(orgId: string, orgName: string) {
		renameOrgId = orgId;
		renameOrgName = orgName;
		renameDialogOpen = true;
	}

	function openDeleteDialog(orgId: string, orgName: string) {
		deleteOrgId = orgId;
		deleteOrgName = orgName;
		deleteDialogOpen = true;
	}

	const isOrgsLoading = $derived(getAllOrgsQuery.isPending);
</script>

<section class="space-y-6">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-medium">Organizations</h2>
		<Button size="sm" onclick={openCreateDialog}>
			<Plus class="size-4" />
			<span class="ml-1">Create</span>
		</Button>
	</div>

	{#if isOrgsLoading}
		<div class="space-y-3">
			<Skeleton class="h-20 w-full" />
			<Skeleton class="h-20 w-full" />
		</div>
	{:else}
		<div class="relative">
			<Search
				class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none"
			/>
			<Input
				type="text"
				placeholder="Search organizations..."
				bind:value={orgSearchQuery}
				class="pl-9"
			/>
		</div>

		{#if orgs.length === 0}
			<p class="text-muted-foreground text-sm">No organizations yet. Create one to get started.</p>
		{:else if filteredOrgs.length === 0}
			<p class="text-muted-foreground text-sm">No organizations match your search.</p>
		{:else}
			<div class="space-y-3">
				{#each filteredOrgs as org (org.id)}
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

							<DropdownMenu.Root>
								<DropdownMenu.Trigger>
									{#snippet child({ props })}
										<Button {...props} variant="ghost" size="icon" title="Organization actions">
											<Settings2 class="size-4" />
										</Button>
									{/snippet}
								</DropdownMenu.Trigger>
								<DropdownMenu.Content align="end" class="w-36">
									{#if org.id !== currentOrg.id}
										<DropdownMenu.Item
											onSelect={() => switchOrg(org.id)}
											disabled={switchOrgMutation.isPending}
										>
											<ArrowRight class="size-4 mr-2" />
											Switch
										</DropdownMenu.Item>
									{/if}
									<DropdownMenu.Item onSelect={() => openRenameDialog(org.id, org.name)}>
										<Pencil class="size-4 mr-2" />
										Rename
									</DropdownMenu.Item>
									{#if org.id !== currentOrg.id}
										<DropdownMenu.Item
											onSelect={() => openDeleteDialog(org.id, org.name)}
											class="text-destructive focus:text-destructive"
										>
											<Trash2 class="size-4 mr-2" />
											Delete
										</DropdownMenu.Item>
									{/if}
								</DropdownMenu.Content>
							</DropdownMenu.Root>
						</div>
					</Card.Root>
				{/each}
			</div>
		{/if}
	{/if}
</section>

<!-- Create Org Dialog -->
<CreateOrgDialog bind:open={createDialogOpen} />

<!-- Rename Org Dialog -->
<RenameOrgDialog bind:open={renameDialogOpen} orgId={renameOrgId} orgName={renameOrgName} />

<!-- Delete Org Dialog -->
<DeleteOrgDialog bind:open={deleteDialogOpen} orgId={deleteOrgId} orgName={deleteOrgName} {orgs} />
