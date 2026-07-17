<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/utils';

	// Section-scoped slide panel with shadcn-style state animations.
	let {
		open = $bindable(false),
		class: className,
		showOverlay = true,
		children
	}: {
		open?: boolean;
		class?: string;
		showOverlay?: boolean;
		children: Snippet;
	} = $props();
</script>

<div class={cn('absolute inset-0 z-30', !open && 'pointer-events-none')} aria-hidden={!open}>
	{#if showOverlay}
		<button
			type="button"
			data-state={open ? 'open' : 'closed'}
			class="absolute inset-0 bg-background/60 data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0 data-closed:opacity-0 duration-200"
			aria-label="Close panel"
			onclick={() => (open = false)}
		></button>
	{/if}

	<aside
		data-state={open ? 'open' : 'closed'}
		class={cn(
			'absolute inset-y-0 right-0 z-10 flex h-full w-full flex-col border-l bg-popover text-popover-foreground shadow-lg sm:w-1/2 data-open:animate-in data-open:slide-in-from-right-10 data-closed:animate-out data-closed:slide-out-to-right-10 duration-200 ease-in-out',
			className,
			!open && 'translate-x-full'
		)}
	>
		{@render children()}
	</aside>
</div>
