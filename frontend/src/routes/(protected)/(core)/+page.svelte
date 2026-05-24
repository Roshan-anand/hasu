<script lang="ts">
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { Search, Trash2 } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { useDeleteProjectMutation } from '@/features/base/mutation.svelte';
	import { useGetAllProjectsQuery } from '@/features/base/query.svelte';
	import CreateProject from './create-project.svelte';
	import { Button } from '@/components/ui/button';

	let searchQuery = $state('');
	const projectsQuery = useGetAllProjectsQuery();
	const deleteProjectMutation = useDeleteProjectMutation();

	const deleteProject = (projectId: string) => {
		if (deleteProjectMutation.isPending) return;
		deleteProjectMutation.mutate({ project_id: projectId });
	};

	const filteredProjects = $derived.by(() => {
		if (!projectsQuery.data) return [];

		const keyword = searchQuery.trim().toLowerCase();
		if (keyword === '') return projectsQuery.data;

		return projectsQuery.data.filter((project) => project.name.toLowerCase().includes(keyword));
	});

	const tempItem = Array.from({ length: 6 });
</script>

<nav class="flex gap-4">
	<div class="flex-1 flex relative">
		<Input
			id="project-search"
			placeholder="Search for projects"
			class="p-2"
			bind:value={searchQuery}
		/>
		<Label class="absolute top-0 right-0 m-1 opacity-75" for="project-search"><Search /></Label>
	</div>
	<CreateProject />
</nav>

<section class="flex-1 p-2">
	{#if projectsQuery.isPending}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each tempItem as _, i (i)}
				<div class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 space-y-3">
					<Skeleton class="h-6 w-3/4" />
					<Skeleton class="h-4 w-1/2" />
				</div>
			{/each}
		</div>
	{:else if projectsQuery.isError}
		<p class="text-red-500">Failed to load projects</p>
	{:else if filteredProjects.length > 0}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each filteredProjects as project (project.id)}
				<div
					class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 hover:shadow-md transition-shadow cursor-pointer relative"
				>
					<a
						href={resolve(`/(protected)/(core)/project/[project_id]`, { project_id: project.id })}
						class="absolute z-10 size-full inset-0 text-transparent"
						title="open project"
					></a>
					<div class="flex items-start justify-between gap-2">
						<div>
							<h3 class="font-semibold text-lg">{project.name}</h3>
							<p class="text-xs uppercase text-muted-foreground">Project</p>
						</div>
						<Button
							variant="destructive"
							size="sm"
							onclick={() => deleteProject(project.id)}
							disabled={deleteProjectMutation.isPending}
							class="z-20"
						>
							{#if deleteProjectMutation.isPending}
								Deleting...
							{:else}
								<Trash2 />
								Delete
							{/if}
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<h3 class="text-muted-foreground size-full flex flex-col items-center justify-center gap-2">
			No projects found
		</h3>
	{/if}
</section>
