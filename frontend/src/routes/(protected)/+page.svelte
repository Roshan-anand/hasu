<script lang="ts">
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Search } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { useGetAllProjectsQuery } from '@/features/base';
	import CreateProject from './create-project.svelte';
	import { ProjectSettings } from '@/components/settings';
	import { DotmSquare } from '@/components/loader';
	import { goto } from '$app/navigation';
	import { Button } from '@/components/ui/button';

	let searchQuery = $state('');
	const projectsQuery = useGetAllProjectsQuery();

	const filteredProjects = $derived.by(() => {
		if (!projectsQuery.data) return [];

		const keyword = searchQuery.trim().toLowerCase();
		if (keyword === '') return projectsQuery.data;

		return projectsQuery.data.filter((project) => project.name.toLowerCase().includes(keyword));
	});
</script>

<nav class="flex gap-4">
	<div class="flex-1 flex items-center border">
		<Label class="opacity-75 border p-1 h-full bg-secondary" for="project-search"><Search /></Label>
		<Input
			id="project-search"
			placeholder="Search for projects"
			class="p-2 h-full"
			bind:value={searchQuery}
		/>
	</div>
	<CreateProject />
</nav>

<section class="flex-1">
	{#if projectsQuery.isPending}
		<div class="size-full flex items-center justify-center">
			<DotmSquare size={65} dotSize={8} />
		</div>
	{:else if projectsQuery.isError}
		<p class="text-red-500">Failed to load projects</p>
	{:else if filteredProjects.length > 0}
		<div class="flex flex-wrap">
			{#each filteredProjects as { id, name } (id)}
				<div
					class="rounded-lg border bg-card text-card-foreground shadow-sm hover:shadow-md transition-shadow cursor-pointer relative h-fit w-[30vw] min-w-25 max-w-75 flex"
				>
					<Button
						variant="none"
						onclick={() =>
							goto(
								resolve('/(protected)/[project]', {
									project: name
								})
							)}
						class="flex-col items-start p-4 py-8 flex-1"
					>
						<h3 class="font-semibold text-lg">{name}</h3>
						<p class="text-xs uppercase text-muted-foreground">Project</p>
					</Button>
					<ProjectSettings projectId={id} {name} />
				</div>
			{/each}
		</div>
	{:else}
		<h3 class="text-muted-foreground size-full flex flex-col items-center justify-center gap-2">
			No projects found
		</h3>
	{/if}
</section>
