<script lang="ts">
	import { useGetOrphanVolumesQuery } from '@/features/base';
	import { VolumeSettings } from '@/components/settings';
	import { DotmSquare } from '@/components/loader';
	import { Input } from '@/components/ui/input';
	import { Button } from '@/components/ui/button';
	import * as Select from '@/components/ui/select';
	import { ArrowUpDown, HardDrive, Search } from '@lucide/svelte';

	type SortDir = 'asc' | 'desc';
	type ServiceTypeFilter = 'all' | 'psql' | 'redis';

	const volumesQuery = useGetOrphanVolumesQuery();

	// Filter state
	let sizeSortDir = $state<SortDir>('asc');
	let typeFilter = $state<ServiceTypeFilter>('all');
	let nameSearch = $state('');

	function toggleSizeSort() {
		sizeSortDir = sizeSortDir === 'asc' ? 'desc' : 'asc';
	}

	// Format size for display
	function formatSize(bytes: number): string {
		if (bytes <= 0) return '—';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		let value = bytes;
		let unitIdx = 0;
		while (value >= 1024 && unitIdx < units.length - 1) {
			value /= 1024;
			unitIdx++;
		}

		console.log(bytes);

		return `${value.toFixed(1)} ${units[unitIdx]}`;
	}

	// Derive human-readable type label
	function typeLabel(t: string): string {
		switch (t) {
			case 'psql':
				return 'Postgres';
			case 'redis':
				return 'Redis';
			default:
				return t;
		}
	}

	// Client-side filtering and sorting
	const filteredVolumes = $derived.by(() => {
		const data = volumesQuery.data;
		if (!data) return [];

		let result = [...data];

		// 1. Filter by service type
		if (typeFilter !== 'all') {
			result = result.filter((v) => v.type === typeFilter);
		}

		// 2. Filter by name (case-insensitive)
		if (nameSearch.trim()) {
			const query = nameSearch.trim().toLowerCase();
			result = result.filter(
				(v) =>
					(v.display_name || v.volume).toLowerCase().includes(query) ||
					v.volume.toLowerCase().includes(query)
			);
		}

		// 3. Sort by size
		result.sort((a, b) => {
			const diff = a.size_bytes - b.size_bytes;
			return sizeSortDir === 'asc' ? diff : -diff;
		});
		return result;
	});

	const volumeCount = $derived(filteredVolumes.length);
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

	<!-- Filter Controls -->
	<div class="flex flex-wrap items-center gap-3">
		<!-- Size sort toggle -->
		<Button variant="outline" size="sm" onclick={toggleSizeSort} class="gap-1.5">
			<ArrowUpDown class="size-3.5" />
			Size
			<span class="text-xs text-muted-foreground">
				{sizeSortDir === 'asc' ? '↑' : '↓'}
			</span>
		</Button>

		<!-- Service type dropdown -->
		<Select.Root
			type="single"
			value={typeFilter}
			onValueChange={(val: string) => (typeFilter = val as ServiceTypeFilter)}
		>
			<Select.Trigger class="w-[140px] h-9">
				{typeFilter === 'all' ? 'All types' : typeFilter === 'psql' ? 'Postgres' : 'Redis'}
			</Select.Trigger>
			<Select.Content>
				<Select.Item value="all">All</Select.Item>
				<Select.Item value="psql">Postgres</Select.Item>
				<Select.Item value="redis">Redis</Select.Item>
			</Select.Content>
		</Select.Root>

		<!-- Name search -->
		<div class="relative min-w-[200px] flex-1 max-w-xs">
			<Search
				class="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none"
			/>
			<Input type="text" placeholder="Search by name…" bind:value={nameSearch} class="pl-8 h-9" />
		</div>
	</div>

	<p class="text-xs text-muted-foreground">
		{volumeCount} orphan volume{volumeCount !== 1 ? 's' : ''}
	</p>

	{#if volumesQuery.isPending}
		<div class="size-full flex items-center justify-center">
			<DotmSquare size={55} dotSize={7} />
		</div>
	{:else if volumesQuery.isError}
		<p class="text-sm text-destructive">Failed to load orphan volumes</p>
	{:else if filteredVolumes.length > 0}
		<div class="flex flex-col gap-3">
			{#each filteredVolumes as volume (volume.id)}
				<div
					class="flex items-center gap-3 rounded-lg border bg-card text-card-foreground shadow-sm px-4 py-3"
				>
					<!-- Volume icon -->
					<div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
						<HardDrive class="size-4 text-muted-foreground" />
					</div>

					<!-- Volume info -->
					<div class="min-w-0 flex-1">
						<p class="text-sm font-medium truncate">
							{volume.display_name || volume.volume}
						</p>
						<div class="flex items-center gap-2 text-xs text-muted-foreground">
							<span class="uppercase">{typeLabel(volume.type)}</span>
							<span aria-hidden="true">·</span>
							<span>{formatSize(volume.size_bytes)}</span>
							<span aria-hidden="true">·</span>
							<span title={new Date(volume.created_at).toLocaleString()}>
								{new Date(volume.created_at).toLocaleDateString()}
							</span>
						</div>
					</div>

					<!-- Settings dropdown (rename + delete) -->
					<VolumeSettings
						volumeId={volume.id}
						volumeName={volume.volume}
						displayName={volume.display_name}
					/>
				</div>
			{/each}
		</div>
	{:else}
		<div class="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
			{nameSearch || typeFilter !== 'all'
				? 'No orphan volumes match the current filters.'
				: 'No orphan volumes found.'}
		</div>
	{/if}
</section>
