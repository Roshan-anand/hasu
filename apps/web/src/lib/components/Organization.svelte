<script lang="ts">
	import { Avatar, AvatarFallback } from '@/components/ui/avatar';
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { Check, ChevronsUpDown, Plus } from '@lucide/svelte';
	import {
		getOrgState,
		GetUserData,
		useGetAllOrgsQuery,
		useSwitchOrgMutation,
		useCreateOrgMutation
	} from '@/features/base';
	import { SidebarMenuButton } from './ui/sidebar';

	const currentOrg = getOrgState();
	const { email } = GetUserData();

	let orgMenuOpen = $state(false);
	let createDialogOpen = $state(false);
	let orgName = $state('');

	const getAllOrgsQuery = useGetAllOrgsQuery();
	const switchOrgMutation = useSwitchOrgMutation();
	const createOrgMutation = useCreateOrgMutation();

	const isOrgListLoading = $derived.by(() => {
		return getAllOrgsQuery.isPending || (getAllOrgsQuery.isFetching && !getAllOrgsQuery.data);
	});

	function switchOrg(orgId: string) {
		if (!orgId || orgId === currentOrg.id || switchOrgMutation.isPending) return;

		switchOrgMutation.mutate(
			{ org_id: orgId },
			{
				onSuccess: () => {
					orgMenuOpen = false;
				}
			}
		);
	}

	function closeCreateOrgDialog() {
		if (createOrgMutation.isPending) return;
		createDialogOpen = false;
	}

	function createOrg() {
		if (orgName.trim().length < 3 || createOrgMutation.isPending) return;

		createOrgMutation.mutate(
			{
				name: orgName.trim()
			},
			{
				onSuccess: () => {
					orgName = '';
					createDialogOpen = false;
				}
			}
		);
	}
</script>

<DropdownMenu.Root bind:open={orgMenuOpen}>
	<SidebarMenuButton
		variant="none"
		class={`relative flex items-center gap-2 p-1 m-0 w-full mx-auto border border-border h-fit hover:bg-sidebar-accent ${orgMenuOpen && 'bg-sidebar-accent'}`}
	>
		<Avatar>
			<AvatarFallback class="rounded-lg">{currentOrg.name.trim()[0] || '?'}</AvatarFallback>
		</Avatar>
		<div class="flex flex-col items-start">
			<p class="truncate font-medium">{currentOrg.name || 'No organization selected'}</p>
			<p class="truncate font-medium opacity-75">{email}</p>
		</div>

		<ChevronsUpDown class="ml-auto" />

		<DropdownMenu.Trigger class="absolute size-full top-0 left-0"></DropdownMenu.Trigger>
	</SidebarMenuButton>
	<DropdownMenu.Content side="bottom" align="start" sideOffset={10} class="w-70 z-50">
		<DropdownMenu.Label>Switch organization</DropdownMenu.Label>
		{#if isOrgListLoading}
			<div class="p-1"><Skeleton class="h-8 w-full" /></div>
		{:else if getAllOrgsQuery.isError}
			<p class="text-destructive px-2 py-1 text-sm">Failed to load organizations</p>
		{:else if getAllOrgsQuery.data && getAllOrgsQuery.data.length > 0}
			<DropdownMenu.Group>
				{#each getAllOrgsQuery.data as org (org.id)}
					<DropdownMenu.Item
						onSelect={() => switchOrg(org.id)}
						disabled={switchOrgMutation.isPending}
					>
						<span class="truncate">{org.name}</span>
						{#if org.id === org.id}
							<Check class="ml-auto" />
						{/if}
					</DropdownMenu.Item>
				{/each}
			</DropdownMenu.Group>
		{:else}
			<p class="text-muted-foreground px-2 py-1 text-sm">No organizations available</p>
		{/if}

		<DropdownMenu.Separator />
		<div class="p-1">
			<Button
				variant="secondary"
				onclick={() => (createDialogOpen = true)}
				disabled={createOrgMutation.isPending}
			>
				<Plus />
				<p>Create</p>
			</Button>
		</div>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<Dialog.Root bind:open={createDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Create Organization</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground">
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
					<Label class="my-1" for="create-org-name">Name</Label>
					<Input
						id="create-org-name"
						placeholder="Organization name"
						bind:value={orgName}
						required
						minlength={3}
						disabled={createOrgMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={closeCreateOrgDialog}
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
