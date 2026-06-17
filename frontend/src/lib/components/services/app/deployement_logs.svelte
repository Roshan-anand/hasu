<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import StreamLogs from '../stream-logs.svelte';

	let { deploymentId, deploymentName }: { deploymentId: string; deploymentName: string } = $props();

	let open = $state(false);
</script>

<Button variant="outline" size="sm" onclick={() => (open = true)}>View</Button>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content class="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
			<Dialog.Title class="text-lg font-semibold">Deployment Logs</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground">
				{deploymentName}
			</Dialog.Description>

			<StreamLogs url={`/api/service/deployment/logs?deployment_id=${deploymentId}`} />
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
