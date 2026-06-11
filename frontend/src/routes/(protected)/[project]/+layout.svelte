<script lang="ts">
	import { useGetAllInstanceQuery } from '@/features/base';

	let { data, children } = $props();
	const project = $derived(data.projectName);

	useGetAllInstanceQuery(() => project);
</script>

{@render children()}
