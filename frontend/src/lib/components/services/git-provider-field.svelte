<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { GitProvidersList } from '@/features/services/const';
	import type { GitProviderKey } from '@/features/services/type';
	import { cn } from '@/utils';
	import Icon from '@iconify/svelte';

	let {
		value,
		onSelect,
		disabled = false
	}: {
		value: GitProviderKey;
		onSelect: (provider: GitProviderKey) => void;
		disabled?: boolean;
	} = $props();
</script>

<div class="flex items-center gap-3 w-full">
	{#each GitProvidersList as [key, provider] (key)}
		<Button
			type="button"
			variant="outline"
			disabled={disabled || provider.listApi === ''}
			onclick={() => onSelect(key)}
			class={cn('flex text-2xl p-4', `${value === key ? 'border-primary' : ''}`)}
		>
			<Icon icon={provider.icon} width="20" height="20" class="size-5" />
			<p>{provider.name}</p>
		</Button>
	{/each}
</div>
