<script lang="ts">
	import { api, axiosErr } from '@/axios';
	import { Avatar, AvatarFallback } from '@/components/ui/avatar';
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { queryClient } from '@/query';
	import { createMutation, createQuery } from '@tanstack/svelte-query';
	import { Check, ChevronsUpDown } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import CreateBtn from './CreateBtn.svelte';
	import { getUserState } from '@/features/global/store.svelte';
	import type { Organization } from '@/features/auth/type';

	const { email, setCurrentOrg, pushOrg, setOrg, currentOrg, orgs } = getUserState();
	let orgMenuOpen = $state(false);
	let createDialogOpen = $state(false);
	let orgName = $state('');

	type SwitchOrgPayload = {
		org_id: string;
	};

	type CreateOrgPayload = {
		name: string;
	};

	const getOrgsQueryKey = () => ['orgs', email] as const;

	// Keeps current org and org list in sync with on-demand query fetch plus switch/create mutations.
	const getAllOrgsQuery = createQuery(() => ({
		queryKey: getOrgsQueryKey(),
		queryFn: () => api.get<Organization[]>('/org').then((res) => res.data),
		enabled: false
	}));

	const switchOrgMutation = createMutation(() => ({
		mutationFn: (payload: SwitchOrgPayload) =>
			api.post<Organization>('/org/switch', payload).then((res) => res.data),
		onSuccess: (org) => {
			setCurrentOrg(org);
			orgMenuOpen = false;
			toast.success('Organization switched successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to switch organization')
	}));

	const createOrgMutation = createMutation(() => ({
		mutationFn: (payload: CreateOrgPayload) =>
			api.post<Organization>('/org', payload).then((res) => res.data),
		onSuccess: (createdOrg) => {
			queryClient.setQueryData(getOrgsQueryKey(), (cachedOrgs: Organization[] | undefined) => {
				if (!cachedOrgs) return [createdOrg];
				if (cachedOrgs.some((org) => org.id === createdOrg.id)) return cachedOrgs;
				return [createdOrg, ...cachedOrgs];
			});

			pushOrg(createdOrg);
			orgName = '';
			createDialogOpen = false;
			toast.success('Organization created successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to create organization')
	}));

	const canCreateOrg = $derived.by(() => orgName.trim().length >= 3);
	const isOrgListLoading = $derived.by(() => {
		return getAllOrgsQuery.isPending || (getAllOrgsQuery.isFetching && !getAllOrgsQuery.data);
	});

	$effect(() => {
		if (!getAllOrgsQuery.data) return;
		setOrg(getAllOrgsQuery.data);
	});

	$effect(() => {
		if (!orgMenuOpen) return;
		void getAllOrgsQuery.refetch();
	});

	function getAvatarText(orgNameValue: string) {
		const [firstWord = ''] = orgNameValue.trim().split(/\s+/);
		return firstWord.slice(0, 1).toUpperCase() || '?';
	}

	function switchOrg(orgId: string) {
		if (!orgId || orgId === currentOrg.id || switchOrgMutation.isPending) return;

		switchOrgMutation.mutate({ org_id: orgId });
	}

	function openCreateOrgDialog() {
		orgMenuOpen = false;
		createDialogOpen = true;
	}

	function closeCreateOrgDialog() {
		if (createOrgMutation.isPending) return;
		createDialogOpen = false;
	}

	function createOrg() {
		if (!canCreateOrg || createOrgMutation.isPending) return;

		createOrgMutation.mutate({
			name: orgName.trim()
		});
	}
</script>

<div class="flex w-full flex-col gap-2">
	<div class="flex w-full items-center gap-2">
		<Avatar>
			<AvatarFallback>{getAvatarText(currentOrg.name)}</AvatarFallback>
		</Avatar>

		<div class="min-w-0 flex-1">
			<p class="truncate font-medium">{currentOrg.name || 'No organization selected'}</p>
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
				{:else if orgs.length > 0}
					<DropdownMenu.Group>
						{#each orgs as org (org.id)}
							<DropdownMenu.Item
								onSelect={() => switchOrg(org.id)}
								disabled={switchOrgMutation.isPending}
							>
								<span class="truncate">{org.name}</span>
								{#if org.id === currentOrg.id}
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
					<Label for="create-org-name">Name</Label>
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
