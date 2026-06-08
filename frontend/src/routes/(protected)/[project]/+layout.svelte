<script lang="ts">
	import { useGetAllInstanceQuery } from '@/features/base';

	let { data, children } = $props();
	const project = $derived(data.projectName);

	useGetAllInstanceQuery(() => project);

	$effect(() => {
		console.log('Project key :', project);
	});
</script>

{@render children()}
