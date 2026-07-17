<script lang="ts">
	import Icon from '@iconify/svelte';
	import { ExternalLink } from '@lucide/svelte';
	import { timeAgo } from '@/utils/time';
	import type { ServiceListApp } from '@/features/services';

	type Props = {
		service: ServiceListApp;
		onclick: () => void;
	};

	let { service, onclick }: Props = $props();

	let replicaLayerCount = $derived(Math.min(Math.max(service.replicas - 1, 0), 3));
	let hasReplicaLayers = $derived(replicaLayerCount > 0);
</script>

<section
	class={['service-card-stack', hasReplicaLayers && 'has-replica-layers', 'size-full']}
	style:--replica-layer-count={replicaLayerCount}
>
	{#if hasReplicaLayers}
		{#each Array(replicaLayerCount) as _, i (i)}
			<!-- Replica layers mimic card-stack scaling below the active card. -->
			<div
				class="replica-layer"
				aria-hidden="true"
				style:--replica-layer={i + 1}
				style:--replica-z={replicaLayerCount - i}
			></div>
		{/each}
	{/if}

	<button
		class="relative z-10 flex w-full flex-col gap-3 rounded-lg border bg-card p-4 text-left transition-shadow hover:shadow-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
		{onclick}
	>
		<div class="flex items-start gap-3">
			<Icon icon="akar-icons:github-fill" class="mt-0.5 size-6 shrink-0" />
			<div class="min-w-0 flex-1">
				<p class="truncate font-semibold text-sm">{service.name}</p>
				<p class="text-xs text-muted-foreground">{timeAgo(service.created_at)}</p>
			</div>
			<div class="flex items-center gap-2 shrink-0">
				<span
					class="rounded-md bg-secondary px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-secondary-foreground"
				>
					Application
				</span>
				<!-- <AppDeletion serviceId={service.id} name={service.name} /> -->
			</div>
		</div>

		<div class="flex flex-col gap-2">
			<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
				<Icon icon="lucide:git-branch" class="size-3 shrink-0" />
				<span class="truncate font-mono">{service.branch_name}</span>
			</div>
			<!-- eslint-disable svelte/no-navigation-without-resolve -->
			<a
				href={'https://' + service.gh_repo_url}
				target="_blank"
				rel="noopener noreferrer"
				onclick={(e) => e.stopPropagation()}
				class="inline-flex items-center gap-1.5 self-start rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted/80 hover:text-foreground no-underline"
			>
				<Icon icon="akar-icons:github-fill" class="size-3" />
				<span class="max-w-40 truncate">{service.gh_repo_name}</span>
				<ExternalLink class="size-3 shrink-0" />
			</a>
		</div>
	</button>
</section>

<style>
	.service-card-stack {
		position: relative;
		isolation: isolate;
		padding-bottom: 0;
	}

	.service-card-stack.has-replica-layers {
		padding-bottom: calc(var(--replica-layer-count) * 10px);
	}

	.replica-layer {
		position: absolute;
		top: calc(var(--replica-layer) * 12px);
		left: 0;
		right: 0;
		height: calc(100% - var(--replica-layer-count) * 10px);
		z-index: var(--replica-z);
		pointer-events: none;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--card);
		transform-origin: top center;
		transform: scale(calc(1 - var(--replica-layer) * 0.035));
		opacity: calc(0.76 - var(--replica-layer) * 0.08);
	}
</style>
