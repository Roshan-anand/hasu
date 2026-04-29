<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import {
		useCreateProjectMutation,
		useDeleteProjectMutation
	} from '@/features/projects/mutation.svelte';
	import { getProjectState } from '@/features/projects/store.svelte';
	import { useProjectsQuery } from '@/features/projects/query.svelte';
	import type { Project } from '@/features/projects/type';
	import { Search, Trash2 } from '@lucide/svelte';
	import * as Dialog from '@/components/ui/dialog';
	import CreateBtn from '@/components/CreateBtn.svelte';
	import { resolve } from '$app/paths';
	import { getUserState } from '@/features/global/store.svelte';

	const { currentOrg } = getUserState();
	const featureState = getProjectState();

	let searchQuery = $state('');

	const query = useProjectsQuery(() => currentOrg.id);
	const createProjectMutation = useCreateProjectMutation();
	const deleteProjectMutation = useDeleteProjectMutation();

	const canCreateProject = $derived.by(() => {
		return featureState.projectName.trim().length >= 3 && currentOrg.id !== '';
	});

	// Opens the create dialog only when an org is selected.
	function openCreateProjectDialog() {
		if (currentOrg.id === '') return;
		featureState.createDialogOpen = true;
	}

	// Closes the create dialog unless a create request is in-flight.
	function closeCreateProjectDialog() {
		if (createProjectMutation.isPending) return;
		featureState.createDialogOpen = false;
	}

	// Triggers project creation with sanitized form values.
	function createProject() {
		if (!canCreateProject || createProjectMutation.isPending) return;

		createProjectMutation.mutate({
			project_name: featureState.projectName.trim(),
			description: featureState.projectDescription.trim()
		});
	}

	// Triggers project deletion for the selected project id.
	function deleteProject(projectId: string) {
		if (deleteProjectMutation.isPending) return;

		deleteProjectMutation.mutate({ id: projectId });
	}

	let projects = $state<Project[]>([]);

	$effect(() => {
		if (!query.data) projects = [];
		else if (!searchQuery) projects = query.data;
		else
			projects = query.data.filter((p) => p.name.toLowerCase().includes(searchQuery.toLowerCase()));
	});

	const tempItem = Array.from({ length: 6 });
</script>

<nav class="flex gap-2">
	<div class="flex-1 flex relative">
		<Input
			id="project-srch"
			placeholder="Search for projects"
			class="p-2"
			bind:value={searchQuery}
		/>
		<Label class="absolute top-0 right-0 m-1 opacity-75" for="project-srch"><Search /></Label>
	</div>
	<CreateBtn onclick={openCreateProjectDialog} disabled={currentOrg.id === ''} />
</nav>

<Dialog.Root bind:open={featureState.createDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Create Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground">
				Add a name and optional description for your project.
			</Dialog.Description>

			<form
				class="mt-4 space-y-4"
				onsubmit={(e) => {
					e.preventDefault();
					createProject();
				}}
			>
				<div class="space-y-1.5">
					<Label for="create-project-name">Name</Label>
					<Input
						id="create-project-name"
						placeholder="Project name"
						bind:value={featureState.projectName}
						required
						minlength={3}
						disabled={createProjectMutation.isPending}
					/>
				</div>

				<div class="space-y-1.5">
					<Label for="create-project-description">Description</Label>
					<textarea
						id="create-project-description"
						class="border-input bg-transparent ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring/50 flex min-h-24 w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-3 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
						placeholder="What is this project for?"
						bind:value={featureState.projectDescription}
						disabled={createProjectMutation.isPending}
					></textarea>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={closeCreateProjectDialog}
						disabled={createProjectMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={!canCreateProject || createProjectMutation.isPending}>
						{createProjectMutation.isPending ? 'Creating...' : 'Create'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<section class="flex-1 p-2">
	{#if query.isPending}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each tempItem as _, i (i)}
				<div
					id={`skeliton-${i}`}
					class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 space-y-3"
				>
					<Skeleton class="h-6 w-3/4" />
					<Skeleton class="h-4 w-1/2" />
				</div>
			{/each}
		</div>
	{:else if query.isError}
		<p class="text-red-500">Failed to load projects</p>
	{:else if projects.length > 0}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each projects as project ((project.id, project.name))}
				<div
					class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 hover:shadow-md transition-shadow cursor-pointer relative"
				>
					<a
						href={resolve('/(core)/project/[id]', {
							id: project.id
						})}
						class="absolute z-10 size-full inset-0 text-transparent"
						title="a"
					></a>
					<div class="flex items-start justify-between gap-2">
						<h3 class="font-semibold text-lg">{project.name}</h3>
						<Button
							variant="destructive"
							size="sm"
							onclick={() => deleteProject(project.id)}
							disabled={deleteProjectMutation.isPending}
							class="z-20"
						>
							{#if deleteProjectMutation.isPending && featureState.deletingProjectId === project.id}
								Deleting...
							{:else}
								<Trash2 />
								Delete
							{/if}
						</Button>
					</div>
					<p class="text-muted-foreground text-sm line-clamp-2">
						{project.description || 'No description'}
					</p>
				</div>
			{/each}
		</div>
	{:else}
		<h3 class="text-muted-foreground size-full flex flex-col justify-center items-center gap-2">
			<span>start a new project</span>
			<CreateBtn onclick={openCreateProjectDialog} disabled={currentOrg.id === ''} />
		</h3>
	{/if}
</section>
