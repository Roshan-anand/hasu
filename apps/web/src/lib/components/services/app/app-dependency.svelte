<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
	import { Input } from '@/components/ui/input';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Skeleton } from '@/components/ui/skeleton';
	import {
		useGetServiceDependenciesQuery,
		useGetDependencyTargetsQuery,
		useGetServiceEnvQuery
	} from '@/features/services';
	import {
		useCreateDependencyMutation,
		useUpdateDependencyMutation,
		useDeleteDependencyMutation
	} from '@/features/services';
	import type { ServiceDependency } from '@/features/services';
	import { AlertTriangle, Info, Pencil, Trash2, X } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import Icon from '@iconify/svelte';
	import { SvelteSet } from 'svelte/reactivity';

	let { serviceID }: { serviceID: string } = $props();

	const depsQuery = useGetServiceDependenciesQuery(() => serviceID);
	const targetsQuery = useGetDependencyTargetsQuery(() => serviceID);
	const envQuery = useGetServiceEnvQuery(() => serviceID);
	// TODO : attention :
	// targerquery filters out empty cols from app service
	// envquery is to validated it user is reusing the env key
	const createDep = useCreateDependencyMutation();
	const updateDep = useUpdateDependencyMutation();
	const deleteDep = useDeleteDependencyMutation();

	let showForm = $state(false);
	let editDialogOpen = $state(false);
	let editingId = $state<string | null>(null);
	let selectedTargetId = $state('');
	let editSelectedTargetId = $state('');
	let editSelectedCol = $state('');
	let editEnvKey = $state('');
	let selectedCol = $state('');
	let envKey = $state('');

	let selectedTarget = $derived(targetsQuery.data?.find((t) => t.id === selectedTargetId));
	let editSelectedTarget = $derived(targetsQuery.data?.find((t) => t.id === editSelectedTargetId));

	let manualEnvKeys = $derived(() => {
		if (!envQuery.data?.env) return new Set<string>();
		const keys = new SvelteSet<string>();
		for (const line of envQuery.data.env) {
			const key = line.split('=')[0]?.trim();
			if (key) keys.add(key);
		}
		return keys;
	});

	let conflicts = $derived(() => {
		if (!depsQuery.data) return [] as ServiceDependency[];
		const manual = manualEnvKeys();
		return depsQuery.data.filter((d) => manual.has(d.env_key));
	});

	let canSubmit = $derived(
		selectedTargetId !== '' &&
			selectedCol !== '' &&
			envKey.trim() !== '' &&
			!createDep.isPending &&
			!updateDep.isPending
	);

	let canEditSubmit = $derived(
		editSelectedTargetId !== '' &&
			editSelectedCol !== '' &&
			editEnvKey.trim() !== '' &&
			!updateDep.isPending
	);

	const resetForm = () => {
		selectedTargetId = '';
		selectedCol = '';
		envKey = '';
	};

	const resetEditForm = () => {
		editingId = null;
		editSelectedTargetId = '';
		editSelectedCol = '';
		editEnvKey = '';
	};

	const cancelForm = () => {
		showForm = false;
		resetForm();
	};

	const openEditForm = (dep: ServiceDependency) => {
		editingId = dep.id;
		editSelectedTargetId = dep.target_service_id;
		editSelectedCol = dep.target_col;
		editEnvKey = dep.env_key;
		editDialogOpen = true;
	};

	const handleSubmit = () => {
		if (!canSubmit) return;
		createDep.mutate(
			{
				source_service_id: serviceID,
				target_service_id: selectedTargetId,
				target_col: selectedCol,
				env_key: envKey.trim()
			},
			{
				onSuccess: () => {
					showForm = false;
					resetForm();
					toast.warning('Changes require a redeploy to take effect.');
				}
			}
		);
	};

	const handleEditSubmit = () => {
		if (!canEditSubmit || !editingId) return;
		updateDep.mutate(
			{
				id: editingId,
				payload: {
					target_service_id: editSelectedTargetId,
					target_col: editSelectedCol,
					env_key: editEnvKey.trim()
				}
			},
			{
				onSuccess: () => {
					editDialogOpen = false;
					resetEditForm();
					toast.warning('Changes require a redeploy to take effect.');
				}
			}
		);
	};

	const handleDelete = (id: string) => {
		deleteDep.mutate(
			{ id, sourceServiceId: serviceID },
			{
				onSuccess: () => {
					toast.warning('Changes require a redeploy to take effect.');
				}
			}
		);
	};

	const colLabel = (col: string) => {
		switch (col) {
			case 'internal_url':
				return 'Internal URL';
			case 'domain':
				return 'Domain';
			case 'db_name':
				return 'DB Name';
			case 'db_user':
				return 'DB User';
			case 'db_password':
				return 'DB Password';
			case 'password':
				return 'Password';
			case 'name':
				return 'Name';
			default:
				return col;
		}
	};

	const typeIcon = (type: string) => {
		switch (type) {
			case 'app':
				return 'icon-park-outline:application';
			case 'psql':
				return 'devicon:postgresql';
			case 'redis':
				return 'devicon:redis';
			default:
				return 'lucide:box';
		}
	};
</script>

<div class="relative border">
	<div class="env-bg absolute pointer-events-none size-full border z-10"></div>

	<!-- Dependency list -->
	{#if depsQuery.isPending}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-8 w-full" />
			<Skeleton class="h-8 w-full" />
			<Skeleton class="h-8 w-full" />
		</div>
	{:else if depsQuery.data && depsQuery.data.length > 0}
		<div class="overflow-hidden rounded-md border font-mono text-xs m-1">
			<div class="flex flex-col divide-y">
				{#each depsQuery.data as dep (dep.id)}
					<div class="group flex items-center gap-3 p-0.5 transition-colors hover:bg-muted/40">
						<div class="flex flex-1 items-center gap-0.5 truncate">
							<Icon icon={typeIcon(dep.target_service_type)} class="h-3.5 w-3.5 shrink-0 mx-2" />
							<span>{dep.env_key}</span>
							<span class="text-muted-foreground/60 px-1"> = </span>
							<span class="text-accent-main-highlight">{'{{'}</span>
							<span>{dep.target_service_name}</span>
							<span class="text-accent-main-highlight font-bold">.</span>
							<span>{colLabel(dep.target_col)}</span>
							<span class="text-accent-main-highlight">}}</span>
						</div>
						<div
							class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity duration-150 group-hover:opacity-100 group-focus-within:opacity-100"
						>
							<Button
								size="sm"
								variant="ghost"
								class="h-7 w-7 p-0"
								onclick={() => openEditForm(dep)}
							>
								<Pencil class="h-3 w-3" />
							</Button>
							<Button
								size="sm"
								variant="ghost"
								class="h-7 w-7 p-0 text-red-500 hover:text-red-600"
								onclick={() => handleDelete(dep.id)}
								disabled={deleteDep.isPending}
							>
								<Trash2 class="h-3 w-3" />
							</Button>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<header class="p-2">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
				<Info class="h-3.5 w-3.5" />
				<span>Inject target service values as environment variables</span>
			</div>
			<Button
				size="sm"
				variant={showForm ? 'secondary' : 'default'}
				onclick={() => {
					showForm = !showForm;
					resetForm();
				}}
			>
				{showForm ? 'Cancel' : 'Add Dependency'}
			</Button>
		</div>
	</header>
	<section class="p-2">
		<!-- Inline create form -->
		{#if showForm}
			<div class="mb-4 flex flex-wrap items-center gap-2">
				<!-- Env key input -->
				<div class="min-w-0 flex-1">
					<Input
						placeholder="DATABASE_URL"
						value={envKey}
						oninput={(e) => (envKey = e.currentTarget.value)}
						class="h-9 font-mono text-xs"
					/>
				</div>

				<!-- Target service select -->
				<div class="min-w-0 h-full" style="flex: 1.5;">
					{#if targetsQuery.isPending}
						<Skeleton class="h-9 w-full" />
					{:else if targetsQuery.data && targetsQuery.data.length > 0}
						<Select.Root
							type="single"
							value={selectedTargetId}
							onValueChange={(v) => {
								selectedTargetId = v;
								selectedCol = '';
							}}
						>
							<Select.Trigger class="h-9 w-full text-xs">
								{#if selectedTarget}
									<div class="flex items-center gap-1.5 truncate">
										<Icon
											icon={typeIcon(selectedTarget.service_type)}
											class="h-3.5 w-3.5 shrink-0"
										/>
										<span class="truncate">{selectedTarget.name}</span>
									</div>
								{:else}
									<span class="text-muted-foreground">Service</span>
								{/if}
							</Select.Trigger>
							<Select.Content>
								{#each targetsQuery.data as target (target.id)}
									<Select.Item value={target.id}>
										<div class="flex items-center gap-2">
											<Icon icon={typeIcon(target.service_type)} class="h-4 w-4" />
											<span>{target.name}</span>
											<span class="text-muted-foreground text-xs">({target.service_type})</span>
										</div>
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					{:else}
						<div
							class="flex h-9 items-center rounded-md border border-dashed px-3 text-xs text-muted-foreground"
						>
							No targets available
						</div>
					{/if}
				</div>

				<!-- Target column select (disabled when no service selected) -->
				<div class="min-w-0" style="flex: 1">
					{#if selectedTarget}
						<Select.Root type="single" value={selectedCol} onValueChange={(v) => (selectedCol = v)}>
							<Select.Trigger class="h-9 w-full text-xs">
								{selectedCol ? colLabel(selectedCol) : 'Column'}
							</Select.Trigger>
							<Select.Content>
								{#each selectedTarget.allowed_cols as col (col)}
									<Select.Item value={col}>
										{colLabel(col)}
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					{:else}
						<div
							class="flex h-9 w-full cursor-not-allowed items-center rounded-md border border-input bg-muted/50 px-3 text-xs text-muted-foreground/50"
						>
							Column
						</div>
					{/if}
				</div>

				<!-- Actions -->
				<Button size="sm" variant="ghost" onclick={cancelForm} class="shrink-0" aria-label="Cancel">
					<X class="h-4 w-4" />
				</Button>
				<Button size="sm" onclick={handleSubmit} disabled={!canSubmit} class="shrink-0">
					{createDep.isPending ? 'Saving…' : 'Create'}
				</Button>
			</div>
		{/if}

		<!-- Conflict warning -->
		{#if depsQuery.data && conflicts().length > 0}
			<div class="mb-3 flex items-start gap-2 rounded-md border bg-red-50 p-3 text-sm text-red-800">
				<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" />
				<div class="flex flex-col gap-1">
					<span class="font-medium">Environment key conflicts detected</span>
					<span
						>The following env keys also exist in manual environment variables and will be
						overridden on deploy:</span
					>
					<ul class="list-disc pl-5">
						{#each conflicts() as dep (dep.id)}
							<li>{dep.env_key}</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}
	</section>
</div>
<!-- Edit dialog -->
<Dialog bind:open={editDialogOpen}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Edit Dependency</DialogTitle>
		</DialogHeader>
		<div class="flex flex-col gap-4 mt-2">
			<div class="flex flex-col gap-2">
				{#if targetsQuery.isPending}
					<Skeleton class="h-10 w-full" />
				{:else if targetsQuery.data && targetsQuery.data.length > 0}
					<Select.Root
						type="single"
						value={editSelectedTargetId}
						onValueChange={(v) => {
							editSelectedTargetId = v;
							editSelectedCol = '';
						}}
					>
						<Select.Trigger class="w-full">
							{editSelectedTarget ? editSelectedTarget.name : 'Select a target service'}
						</Select.Trigger>
						<Select.Content>
							{#each targetsQuery.data as target (target.id)}
								<Select.Item value={target.id}>
									<div class="flex items-center gap-2">
										<Icon icon={typeIcon(target.service_type)} class="h-4 w-4" />
										<span>{target.name}</span>
										<span class="text-muted-foreground text-xs">({target.service_type})</span>
									</div>
								</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				{:else}
					<p class="text-sm text-muted-foreground">
						No available target services in this instance.
					</p>
				{/if}
			</div>

			{#if editSelectedTarget}
				<div class="flex flex-col gap-2">
					<Select.Root
						type="single"
						value={editSelectedCol}
						onValueChange={(v) => (editSelectedCol = v)}
					>
						<Select.Trigger class="w-full">
							{editSelectedCol ? colLabel(editSelectedCol) : 'Select a column'}
						</Select.Trigger>
						<Select.Content>
							{#each editSelectedTarget.allowed_cols as col (col)}
								<Select.Item value={col}>
									{colLabel(col)}
								</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
			{/if}

			<div class="flex flex-col gap-2">
				<Input
					placeholder="e.g. DATABASE_URL"
					value={editEnvKey}
					oninput={(e) => (editEnvKey = e.currentTarget.value)}
				/>
				<p class="text-xs text-muted-foreground">
					Alphanumeric and underscores only. Must start with a letter or underscore.
				</p>
			</div>

			<Button onclick={handleEditSubmit} disabled={!canEditSubmit} class="mt-2">
				{updateDep.isPending ? 'Saving…' : 'Update Dependency'}
			</Button>
		</div>
	</DialogContent>
</Dialog>
