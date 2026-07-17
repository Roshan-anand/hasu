<!-- this componenet is fully vibe coded, some logics may not make sence-->
<script lang="ts">
	import { graphlib, layout } from '@dagrejs/dagre';
	import { useGetDependencyGraphQuery } from '@/features/services';
	import type { ServiceListResponse } from '@/features/services';
	import { DotmSquare } from '@/components/loader';
	import AppServiceCard from './AppServiceCard.svelte';
	import PsqlServiceCard from './PsqlServiceCard.svelte';

	type Props = {
		services: ServiceListResponse[];
		onNodeClick: (service: ServiceListResponse) => void;
	};

	let { services, onNodeClick }: Props = $props();

	const NODE_WIDTH = 250;
	const NODE_HEIGHT = 180;

	const query = useGetDependencyGraphQuery();

	// ── dagre layout ─────────────────────────────────────────────
	let layoutResult = $derived.by(() => {
		if (!query.data?.nodes.length) return null;

		const g = new graphlib.Graph()
			.setGraph({ rankdir: 'LR', nodesep: 48, ranksep: 100, marginx: 40, marginy: 40 })
			.setDefaultEdgeLabel(() => ({}));

		for (const node of query.data.nodes) {
			g.setNode(node.id, { ...node, width: NODE_WIDTH, height: NODE_HEIGHT });
		}

		for (const edge of query.data.edges) {
			g.setEdge(edge.source, edge.target, edge);
		}

		layout(g);

		const graphInfo = g.graph();

		const nodes = query.data.nodes.map((node) => {
			const n = g.node(node.id)!;
			return { ...node, x: n.x as number, y: n.y as number };
		});

		const edges = query.data.edges.map((edge) => {
			const e = g.edge(edge.source, edge.target)!;
			return { ...edge, points: (e.points ?? []) as { x: number; y: number }[] };
		});

		return {
			width: (graphInfo.width as number) + 80,
			height: (graphInfo.height as number) + 80,
			nodes,
			edges
		};
	});

	// ── edge path ────────────────────────────────────────────────
	function edgePath(points: { x: number; y: number }[]) {
		if (points.length === 0) return '';
		if (points.length === 1) return `M ${points[0].x} ${points[0].y}`;

		return points
			.map((point, index) => {
				if (index === 0) return `M ${point.x} ${point.y}`;
				const previous = points[index - 1];
				const midX = (previous.x + point.x) / 2;
				return `C ${midX} ${previous.y}, ${midX} ${point.y}, ${point.x} ${point.y}`;
			})
			.join(' ');
	}

	// ── hover state ──────────────────────────────────────────────
	let hoveredEdgeKey = $state<string | null>(null);

	// ── pan / zoom ───────────────────────────────────────────────
	let zoom = $state(0.4);
	let panX = $state(0);
	let panY = $state(0);
	let isPanning = $state(false);
	let panStart = $state({ x: 0, y: 0 });
	let panStartOffset = $state({ x: 0, y: 0 });
	let containerEl: HTMLDivElement | null = $state(null);
	let hasCentered = $state(false);

	const MIN_ZOOM = 0.25;
	const MAX_ZOOM = 3;

	function handleWheel(event: WheelEvent) {
		event.preventDefault();
		const delta = -event.deltaY * 0.001;
		const newZoom = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom + delta * zoom));
		zoom = newZoom;
	}

	function handlePanStart(event: MouseEvent) {
		if (event.button !== 0) return;
		isPanning = true;
		panStart = { x: event.clientX, y: event.clientY };
		panStartOffset = { x: panX, y: panY };
	}

	function handlePanMove(event: MouseEvent) {
		if (!isPanning) return;
		panX = panStartOffset.x + (event.clientX - panStart.x);
		panY = panStartOffset.y + (event.clientY - panStart.y);
	}

	function handlePanEnd() {
		isPanning = false;
	}

	function centerView() {
		if (!containerEl || !layoutResult) return;
		const { clientWidth, clientHeight } = containerEl;
		panX = (clientWidth - layoutResult.width * zoom) / 2;
		panY = (clientHeight - layoutResult.height * zoom) / 2;
	}

	$effect(() => {
		if (layoutResult && containerEl && !hasCentered) {
			centerView();
			hasCentered = true;
		}
	});

	export function resetView() {
		zoom = 0.4;
		centerView();
	}

	// ── node click resolution ────────────────────────────────────
	function getServiceForNode(nodeId: string): ServiceListResponse | undefined {
		return services.find((s) => s.id === nodeId);
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
	class="graph-container relative w-full h-full max-h-[80vh] overflow-hidden rounded-lg border bg-background"
	bind:this={containerEl}
	role="application"
	aria-label="Service dependency graph. Drag to pan, scroll to zoom."
	onwheel={handleWheel}
	onmousedown={handlePanStart}
	onmousemove={handlePanMove}
	onmouseup={handlePanEnd}
	onmouseleave={handlePanEnd}
>
	{#if query.isPending}
		<div class="flex size-full items-center justify-center">
			<DotmSquare class="size-12 text-muted-foreground" />
		</div>
	{:else if query.isError}
		<div class="flex size-full items-center justify-center">
			<p class="text-destructive text-sm font-medium">Failed to load dependency graph</p>
		</div>
	{:else if !layoutResult || layoutResult.nodes.length === 0}
		<div class="flex size-full items-center justify-center">
			<p class="text-muted-foreground text-sm">No services to display</p>
		</div>
	{:else}
		<svg
			viewBox="0 0 {layoutResult.width} {layoutResult.height}"
			class="block"
			style="transform: scale({zoom}) translate({panX / zoom}px, {panY /
				zoom}px); transform-origin: 0 0; cursor: {isPanning ? 'grabbing' : 'grab'};"
		>
			<!-- Edges -->
			{#each layoutResult.edges as edge, i (edge.source + '-' + edge.target + i)}
				{@const edgeKey = edge.source + '-' + edge.target}
				{@const isHovered = hoveredEdgeKey === edgeKey}
				<path
					d={edgePath(edge.points)}
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
					class="transition-colors duration-150 {isHovered
						? 'text-primary'
						: 'text-muted-foreground'}"
					role="button"
					tabindex="-1"
					onmouseenter={() => (hoveredEdgeKey = edgeKey)}
					onmouseleave={() => (hoveredEdgeKey = null)}
				/>
			{/each}

			<!-- Nodes -->
			{#each layoutResult.nodes as node, i (node.id + i)}
				{@const matchingService = getServiceForNode(node.id)}
				<g transform="translate({node.x - NODE_WIDTH / 2}, {node.y - NODE_HEIGHT / 2})">
					<foreignObject x="0" y="0" width={NODE_WIDTH} height={NODE_HEIGHT}>
						<div class="flex items-center">
							{#if matchingService}
								{#if matchingService.type === 'app'}
									<AppServiceCard
										service={matchingService}
										onclick={() => onNodeClick(matchingService)}
									/>
								{:else if matchingService.type === 'psql'}
									<PsqlServiceCard
										service={matchingService}
										onclick={() => onNodeClick(matchingService)}
									/>
								{:else}
									<div
										class="flex w-full items-center justify-center rounded-lg border bg-card p-2"
									>
										<p class="truncate text-sm text-muted-foreground">{node.name}</p>
									</div>
								{/if}
							{:else}
								<div class="flex w-full items-center justify-center rounded-lg border bg-card p-2">
									<p class="truncate text-sm text-muted-foreground">{node.name}</p>
								</div>
							{/if}
						</div>
					</foreignObject>
				</g>
			{/each}
		</svg>
	{/if}
</div>

<style>
	.graph-container {
		background-image: radial-gradient(circle, var(--stroke) 1px, transparent 1px);
		background-size: 24px 24px;
	}
</style>
