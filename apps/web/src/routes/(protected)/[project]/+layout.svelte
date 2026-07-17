<script lang="ts">
	import { useGetAllInstanceQuery } from '@/features/base';
	import { setBaseState } from '@/features/base';

	let { data, children } = $props();
	const project = $derived(data.projectName);

	useGetAllInstanceQuery(() => project);
	setBaseState();
</script>

{@render children()}
