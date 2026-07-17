<script lang="ts">
	import { Play, Settings2, Square } from '@lucide/svelte';
	import { Button } from '@/components/ui/button';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import {
		useRedeployPsqlServiceMutation,
		useStopPredefServiceMutation,
		useStartPredefServiceMutation
	} from '@/features/services';
	import { useGetPsqlServiceDetailsQuery } from '@/features/services';
	import { DbDeletion } from '@/components/conformation';

	let { serviceID, name }: { serviceID: string; name: string } = $props();

	const serviceQuery = useGetPsqlServiceDetailsQuery(() => serviceID);
	const redeployPsqlService = useRedeployPsqlServiceMutation();
	const stopService = useStopPredefServiceMutation(() => serviceID);
	const startService = useStartPredefServiceMutation(() => serviceID);

	const isRunning = $derived(serviceQuery.data?.status === 'running');
	const isPaused = $derived(serviceQuery.data?.status === 'paused');
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="outline" size="icon" class="size-8">
				<Settings2 class="size-4" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-44">
		{#if isRunning}
			<Button
				variant="outline"
				class="border-amber-500/35 bg-amber-500/10 text-amber-800 hover:bg-amber-500/15 hover:text-amber-900 dark:border-amber-400/30 dark:bg-amber-400/10 dark:text-amber-200 dark:hover:bg-amber-400/15 dark:hover:text-amber-100 w-full"
				onclick={() => stopService.mutate()}
				disabled={stopService.isPending}
			>
				{#if stopService.isPending}
					Stopping...
				{:else}
					<Square class="mr-1.5 h-4 w-4" />
					Stop
				{/if}
			</Button>
		{:else if isPaused}
			<Button
				variant="outline"
				class="border-emerald-500/35 bg-emerald-500/10 text-emerald-800 hover:bg-emerald-500/15 hover:text-emerald-900 dark:border-emerald-400/30 dark:bg-emerald-400/10 dark:text-emerald-200 dark:hover:bg-emerald-400/15 dark:hover:text-emerald-100"
				onclick={() => startService.mutate()}
				disabled={startService.isPending}
			>
				{#if startService.isPending}
					Starting...
				{:else}
					<Play class="mr-1.5 h-4 w-4" />
					Start
				{/if}
			</Button>
		{/if}
		<Button
			variant="outline"
			onclick={() => redeployPsqlService.mutate({ service_id: serviceID })}
			disabled={redeployPsqlService.isPending || isPaused}
			class="w-full my-1"
		>
			{redeployPsqlService.isPending ? 'Redeploying...' : 'Redeploy'}
		</Button>
		<DropdownMenu.Separator />
		<DbDeletion serviceId={serviceID} {name} />
	</DropdownMenu.Content>
</DropdownMenu.Root>
