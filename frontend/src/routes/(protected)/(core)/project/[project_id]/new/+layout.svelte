<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { Button } from '@/components/ui/button';
	import { MoveLeft } from '@lucide/svelte';

	let { children, data } = $props();
	const projectId = $derived(data.project_id);
</script>

<header class="flex justify-between items-center p-2">
	<Button
		variant="link"
		onclick={() =>
			goto(resolve('/(protected)/(core)/project/[project_id]', { project_id: projectId }))}
	>
		<MoveLeft class="size-10 text-foreground" />
	</Button>
</header>
<main class="p-2 flex-1 flex flex-col">
	{@render children()}
</main>
