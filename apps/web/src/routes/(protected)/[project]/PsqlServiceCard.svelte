<script lang="ts">
	import Icon from '@iconify/svelte';
	import { timeAgo } from '@/utils/time';
	import type { ServiceListPSQL } from '@/features/services';

	type Props = {
		service: ServiceListPSQL;
		onclick: () => void;
	};

	let { service, onclick }: Props = $props();
</script>

<section class="flex flex-col size-full">
	<button
		class="flex-1 flex flex-col justify-between gap-3 rounded-lg border bg-card p-4 text-left transition-shadow hover:shadow-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
		{onclick}
	>
		<div class="flex items-start gap-3">
			<Icon icon="logos:postgresql" class="mt-0.5 size-6 shrink-0" />
			<div class="min-w-0 flex-1">
				<p class="truncate font-semibold text-sm">{service.name}</p>
				<p class="text-xs text-muted-foreground">{timeAgo(service.created_at)}</p>
			</div>
			<div class="flex items-center gap-2 shrink-0">
				<span
					class="rounded-md bg-secondary px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-secondary-foreground"
				>
					PostgreSQL
				</span>
			</div>
		</div>

		<div class="flex flex-col gap-2">
			<div class="flex items-center gap-2">
				{#if service.status === 'running'}
					<div class="flex items-center gap-1.5">
						<span class="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></span>
						<span class="text-xs font-medium text-emerald-600 dark:text-emerald-400">Running</span>
					</div>
				{:else if service.status === 'paused'}
					<div class="flex items-center gap-1.5">
						<span class="h-2 w-2 rounded-full bg-amber-500"></span>
						<span class="text-xs font-medium text-amber-600 dark:text-amber-400">Paused</span>
					</div>
				{:else}
					<span class="text-xs text-muted-foreground capitalize">{service.status}</span>
				{/if}
			</div>
		</div>
	</button>

	<!-- Volume card uses the same stack treatment as replicas, but with one fixed backing layer. -->
	<div
		class="volume-layer border rounded-lg scale-99 flex justify-center gap-3 py-1"
		aria-hidden="true"
	>
		<Icon icon="mdi:storage" class="size-4 shrink-0 text-muted-foreground" />
		<p class="truncate font-medium text-xs text-muted-foreground">{service.volume}</p>
	</div>
</section>

<style>
	.volume-layer {
		pointer-events: none;
		background: color-mix(in oklab, var(--card) 72%, var(--background));
		opacity: 0.58;
	}
</style>
