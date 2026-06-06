<script lang="ts">
	import { useGetOrphanVolumesQuery } from '@/features/base/query.svelte';
	import VolumeDeletion from '@/components/conformation/volume-deletion.svelte';
	import { DotmSquare } from '@/components/loader';

	const volumesQuery = useGetOrphanVolumesQuery();

	const volumeCount = $derived.by(() => volumesQuery.data?.length ?? 0);
</script>

<section class="flex flex-1 flex-col gap-4 p-2">
	<header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="space-y-1">
			<h2 class="text-lg font-semibold">Orphan Volumes</h2>
			<p class="text-xs text-muted-foreground">
				Preserved data volumes that are no longer attached to a running service.
			</p>
		</div>
	</header>

	<p class="text-xs text-muted-foreground">{volumeCount} orphan volumes</p>

	{#if volumesQuery.isPending}
		<div class="size-full flex items-center justify-center">
			<DotmSquare size={55} dotSize={7} />
		</div>
	{:else if volumesQuery.isError}
		<p class="text-sm text-destructive">Failed to load orphan volumes</p>
	{:else if volumesQuery.data && volumesQuery.data.length > 0}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
			{#each volumesQuery.data as volume (volume.id)}
				<div class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 space-y-3">
					<div class="flex items-start justify-between gap-2">
						<div class="min-w-0">
							<p class="text-sm font-medium break-all">{volume.volume}</p>
							<p class="text-xs uppercase text-muted-foreground">{volume.type}</p>
						</div>
						<VolumeDeletion volume={volume.volume} />
					</div>
					<p class="text-xs text-muted-foreground">
						Created {new Date(volume.created_at).toLocaleString()}
					</p>
				</div>
			{/each}
		</div>
	{:else}
		<div class="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
			No orphan volumes found for this selection.
		</div>
	{/if}
</section>
