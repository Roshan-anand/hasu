<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { gitProviders } from '@/features/services/const';
	import type { GitProviderKey, GitProviderOption } from '@/features/services/type';
	import Icon from '@iconify/svelte';

	let {
		value,
		onSelect,
		disabled = false
	}: {
		value: GitProviderKey;
		onSelect: (provider: GitProviderOption) => void;
		disabled?: boolean;
	} = $props();
</script>

<div class="flex items-center gap-3 w-full">
	{#each gitProviders as provider (provider.key)}
		<Button
			type="button"
			variant="outline"
			disabled={disabled || provider.api === ''}
			onclick={() => onSelect(provider)}
			class={`flex-1 ${value === provider.key ? 'border-primary' : ''}`}
		>
			<Icon icon={provider.icon} width="20" height="20" />
			<p>{provider.name}</p>
		</Button>
	{/each}
</div>
