<script lang="ts">
	import { Button } from '@/components/ui/button';
	import PsqlForm from './psql-form.svelte';
	import RedisForm from './redis-form.svelte';
	import Icon from '@iconify/svelte';

	const { data } = $props();

	type DbType = 'psql' | 'redis' | null;

	let selectedDb = $state<DbType>('psql');
</script>

<section class="mx-auto w-full max-w-3xl p-4 md:p-6 flex flex-col gap-2">
	<h1>New Database Service</h1>

	<div class="flex gap-4">
		<Button variant={selectedDb === 'psql' ? 'default' : 'outline'} class="gap-3 text-base" onclick={() => (selectedDb = 'psql')}>
			<Icon icon="logos:postgresql" class="mt-0.5 size-6 shrink-0" />
			<span>PostgreSQL</span>
		</Button>
		<Button variant={selectedDb === 'redis' ? 'default' : 'outline'} class="gap-3 text-base" onclick={() => (selectedDb = 'redis')}>
			<Icon icon="logos:redis" class="mt-0.5 size-6 shrink-0" />
			<span>Redis</span>
		</Button>
	</div>
	{#if selectedDb === 'psql'}
		<PsqlForm {data} />
	{:else if selectedDb === 'redis'}
		<RedisForm {data} />
	{/if}
</section>
