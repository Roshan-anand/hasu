<script lang="ts">
	import { MATRIX_SIZE, rowMajorIndex } from './dotmatrix-utils.js';

	interface Props {
		size?: number;
		dotSize?: number;
		speed?: number;
		ariaLabel?: string;
		animated?: boolean;
		hoverAnimated?: boolean;
		class?: string;
		color?: string;
	}

	let {
		size = 36,
		dotSize = 5,
		speed = 1.45,
		ariaLabel = 'Loading',
		animated = true,
		hoverAnimated = false,
		class: className = '',
		color = '#e56f00'
	}: Props = $props();

	// ---------------------------------------------------------------------------
	// Clockwise perimeter path — one closed loop around the 5×5 grid edge (16 cells)
	// ---------------------------------------------------------------------------
	const PERIMETER_PATH: readonly number[] = [
		rowMajorIndex(0, 0),
		rowMajorIndex(0, 1),
		rowMajorIndex(0, 2),
		rowMajorIndex(0, 3),
		rowMajorIndex(0, 4),
		rowMajorIndex(1, 4),
		rowMajorIndex(2, 4),
		rowMajorIndex(3, 4),
		rowMajorIndex(4, 4),
		rowMajorIndex(4, 3),
		rowMajorIndex(4, 2),
		rowMajorIndex(4, 1),
		rowMajorIndex(4, 0),
		rowMajorIndex(3, 0),
		rowMajorIndex(2, 0),
		rowMajorIndex(1, 0)
	];

	const LOOP_LEN = PERIMETER_PATH.length; // 16

	// Brightness ramps for the two chasing tails
	const TAIL_BRIGHT = [1, 0.82, 0.64, 0.46, 0.3, 0.18] as const;
	const BACK_TAIL_BRIGHT = [0.38, 0.3, 0.22, 0.14] as const;

	// Opacity constants
	const BASE_OPACITY = 0.08;
	const TWIST_INNER_OPACITY = 0.52;
	const SEAM_PULSE_OPACITY = 0.55;
	const IDLE_RING_OPACITY = 0.48;

	// Corner steps → one cell "inside" the strip (half-twist cue)
	const TWIST_INNER_BY_HEAD_STEP: ReadonlyMap<number, number> = new Map([
		[0, rowMajorIndex(1, 1)],
		[4, rowMajorIndex(1, 3)],
		[8, rowMajorIndex(3, 3)],
		[12, rowMajorIndex(3, 1)]
	]);

	const CENTER_CELL = rowMajorIndex(2, 2);

	// ---------------------------------------------------------------------------
	// Layout helpers
	// ---------------------------------------------------------------------------
	const FULL_INDEXES = Array.from({ length: MATRIX_SIZE * MATRIX_SIZE }, (_, i) => i);

	const gap = $derived(Math.max(1, Math.floor((size - dotSize * MATRIX_SIZE) / (MATRIX_SIZE - 1))));
	const matrixSpan = $derived(size);

	// ---------------------------------------------------------------------------
	// prefers-reduced-motion
	// ---------------------------------------------------------------------------
	let reducedMotion = $state(false);

	$effect(() => {
		const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
		reducedMotion = mq.matches;
		const handler = () => {
			reducedMotion = mq.matches;
		};
		mq.addEventListener('change', handler);
		return () => mq.removeEventListener('change', handler);
	});

	// ---------------------------------------------------------------------------
	// Phase management (idle / loadingRipple / collapse / hoverRipple)
	// ---------------------------------------------------------------------------
	type Phase = 'idle' | 'loadingRipple' | 'collapse' | 'hoverRipple';

	const autoRun = $derived(animated && !hoverAnimated && !reducedMotion);
	let hoverPhase = $state<Phase>('idle');
	let hoverGen = 0;
	let timeoutIds: ReturnType<typeof setTimeout>[] = [];

	function clearTimers() {
		for (const id of timeoutIds) clearTimeout(id);
		timeoutIds = [];
	}

	function onMouseEnter() {
		if (!hoverAnimated || autoRun) return;
		clearTimers();
		hoverGen++;
		hoverPhase = 'collapse';
		const collapseMs = Math.max(1, Math.round(300 / (speed > 0 ? speed : 1)));
		const gen = hoverGen;
		const id = setTimeout(() => {
			if (hoverGen !== gen) return;
			hoverPhase = 'hoverRipple';
		}, collapseMs);
		timeoutIds.push(id);
	}

	function onMouseLeave() {
		if (!hoverAnimated || autoRun) return;
		hoverGen++;
		clearTimers();
		hoverPhase = 'idle';
	}

	const phase: Phase = $derived(autoRun ? 'loadingRipple' : hoverAnimated ? hoverPhase : 'idle');

	// Cleanup timeouts on unmount
	$effect(() => {
		return () => clearTimers();
	});

	// ---------------------------------------------------------------------------
	// Stepped cycle — drives the head position along the perimeter loop
	// ---------------------------------------------------------------------------
	let headStep = $state(0);
	const safeSpeed = $derived(speed > 0 ? speed : 1);
	const cycleMsBase = 1600;
	const stepMs = $derived(cycleMsBase / safeSpeed / LOOP_LEN);
	const cycleMs = $derived(stepMs * LOOP_LEN);

	$effect(() => {
		if (reducedMotion || phase === 'idle') {
			headStep = 0;
			return;
		}

		let startMs = 0;
		let lastStep = 0;
		let rafId = 0;

		const tick = (now: number) => {
			if (startMs === 0) startMs = now;
			const elapsed = Math.max(0, now - startMs);
			const nextStep = Math.floor((elapsed % cycleMs) / stepMs) % LOOP_LEN;
			if (nextStep !== lastStep) {
				lastStep = nextStep;
				headStep = nextStep;
			}
			rafId = requestAnimationFrame(tick);
		};

		rafId = requestAnimationFrame(tick);
		return () => cancelAnimationFrame(rafId);
	});

	// ---------------------------------------------------------------------------
	// Per-cell opacity resolver
	// ---------------------------------------------------------------------------
	function opacityFromTail(distance: number, tail: readonly number[]): number {
		if (distance < 0 || distance >= tail.length) return 0;
		return tail[distance]!;
	}

	function pathStepForCellIndex(cellIndex: number): number {
		return PERIMETER_PATH.indexOf(cellIndex);
	}

	function resolveOpacity(index: number): number {
		const onLoop = pathStepForCellIndex(index);
		const backHead = (headStep + Math.floor(LOOP_LEN / 2)) % LOOP_LEN;

		// Idle / reduced motion — static display
		if (reducedMotion || phase === 'idle') {
			if (onLoop >= 0) return IDLE_RING_OPACITY;
			if (index === CENTER_CELL) return 0.22;
			return BASE_OPACITY;
		}

		let opacity = BASE_OPACITY;

		// Forward + backward chasing tails along the perimeter
		if (onLoop >= 0) {
			const forward = (headStep - onLoop + LOOP_LEN) % LOOP_LEN;
			const alongBack = (backHead - onLoop + LOOP_LEN) % LOOP_LEN;
			opacity = Math.max(
				opacity,
				opacityFromTail(forward, TAIL_BRIGHT),
				opacityFromTail(alongBack, BACK_TAIL_BRIGHT)
			);
		}

		// Twist inner cell at corner steps
		const twistInner = TWIST_INNER_BY_HEAD_STEP.get(headStep);
		if (twistInner === index) {
			opacity = Math.max(opacity, TWIST_INNER_OPACITY);
		}

		// Center seam pulse every 4 steps
		if (index === CENTER_CELL && headStep % 4 === 0) {
			opacity = Math.max(opacity, SEAM_PULSE_OPACITY);
		}

		return Math.min(1, opacity);
	}

	// ---------------------------------------------------------------------------
	// Grid geometry (CSS custom properties)
	// ---------------------------------------------------------------------------
	const gridStyle = $derived(`--dmx-dot-size: ${dotSize}px; color: ${color};`);
</script>

<div
	class="dmx-root {className}"
	style="width: {matrixSpan}px; height: {matrixSpan}px; {gridStyle}"
	role="status"
	aria-live="polite"
	aria-label={ariaLabel}
	onmouseenter={onMouseEnter}
	onmouseleave={onMouseLeave}
>
	<div class="dmx-grid" style="gap: {gap}px;">
		{#each FULL_INDEXES as index (index)}
			{@const opacity = resolveOpacity(index)}
			<span
				class="dmx-dot"
				style="width: {dotSize}px; height: {dotSize}px; opacity: {opacity};"
				aria-hidden="true"
			></span>
		{/each}
	</div>
</div>

<style>
	.dmx-root {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		vertical-align: middle;
	}

	.dmx-grid {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		grid-template-rows: repeat(5, minmax(0, 1fr));
	}

	.dmx-dot {
		display: block;
		border-radius: 999px;
		background: var(--dmx-dot-fill, currentColor);
		will-change: opacity;
	}

	@media (prefers-reduced-motion: reduce) {
		.dmx-dot {
			transition: none !important;
		}
	}
</style>
