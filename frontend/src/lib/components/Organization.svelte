<script lang="ts">
	import { Avatar, AvatarFallback } from '@/components/ui/avatar';
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { Check, ChevronsUpDown } from '@lucide/svelte';
	import CreateBtn from './CreateBtn.svelte';
	import { GetUserData } from '@/features/global/query';
	import { useGetAllOrgsQuery } from '@/features/base/query.svelte';
	import { useSwitchOrgMutation, useCreateOrgMutation } from '@/features/base/mutation.svelte';

	const user = GetUserData();

	const { currentOrgName, currentOrgID } = $derived.by(() => {
		console.log('triggerd derived for org data');
		if (switchOrgMutation.isSuccess) {
			const { org_id, org_name } = GetUserData();
			return {
				currentOrgID: org_id,
				currentOrgName: org_name
			};
		}

		return {
			currentOrgName: user.org_name,
			currentOrgID: user.org_id
		};
	});

	let orgMenuOpen = $state(false);
	let createDialogOpen = $state(false);
	let orgName = $state('');

	// Keeps current org and org list in sync with on-demand query fetch plus switch/create mutations.
	const getAllOrgsQuery = useGetAllOrgsQuery();
	const switchOrgMutation = useSwitchOrgMutation();
	const createOrgMutation = useCreateOrgMutation();

	const canCreateOrg = $derived.by(() => orgName.trim().length >= 3);
	const isOrgListLoading = $derived.by(() => {
		return getAllOrgsQuery.isPending || (getAllOrgsQuery.isFetching && !getAllOrgsQuery.data);
	});

	// TODO : this cuase inifinit refetch
	// $effect(() => {
	// 	if (orgMenuOpen) {
	// 		console.log('trigger reFetching org list...');
	// 		void getAllOrgsQuery.refetch();
	// 	}
	// });

	function getAvatarText(orgNameValue: string) {
		const [firstWord = ''] = orgNameValue.trim().split(/\s+/);
		return firstWord.slice(0, 1).toUpperCase() || '?';
	}

	function switchOrg(orgId: string) {
		if (!orgId || orgId === currentOrgID || switchOrgMutation.isPending) return;

		switchOrgMutation.mutate(
			{ org_id: orgId },
			{
				onSuccess: () => {
					orgMenuOpen = false;
				}
			}
		);
	}

	function openCreateOrgDialog() {
		// orgMenuOpen = false;
		createDialogOpen = true;
	}

	function closeCreateOrgDialog() {
		if (createOrgMutation.isPending) return;
		createDialogOpen = false;
	}

	function createOrg() {
		if (!canCreateOrg || createOrgMutation.isPending) return;

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

<div class="flex w-full flex-col gap-2">
	<div class="flex w-full items-center gap-2">
		<Avatar>
			<AvatarFallback>{getAvatarText(currentOrgName)}</AvatarFallback>
		</Avatar>

		<div class="min-w-0 flex-1">
			<p class="truncate font-medium">{currentOrgName || 'No organization selected'}</p>
		</div>

		<DropdownMenu.Root bind:open={orgMenuOpen}>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Button variant="outline" size="sm" disabled={switchOrgMutation.isPending} {...props}>
						<ChevronsUpDown />
					</Button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content side="left" align="start" class="w-64">
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
								{#if org.id === currentOrgID}
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
					<CreateBtn onclick={openCreateOrgDialog} disabled={createOrgMutation.isPending} />
				</div>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>
</div>

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
					<Button type="submit" disabled={!canCreateOrg || createOrgMutation.isPending}>
						{createOrgMutation.isPending ? 'Creating...' : 'Create'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
