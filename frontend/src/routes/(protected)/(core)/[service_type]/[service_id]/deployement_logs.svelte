<script lang="ts">
	import { browser } from '$app/environment';
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';

	let { deploymentId, deploymentName }: { deploymentId: string; deploymentName: string } = $props();

	let open = $state(false);
	let logs = $state<string[]>([]);
	let streamState = $state<'connecting' | 'connected' | 'closed'>('closed');
	let eventSource: EventSource | null = null;

	// append incoming log lines, and always close the stream when the dialog closes.
	function closeStream() {
		if (!eventSource) return;
		eventSource.close();
		eventSource = null;
		streamState = 'closed';
	}

	function connectLogs() {
		if (!browser || deploymentId === '') return;

		closeStream();
		logs = [];
		streamState = 'connecting';

		const url = `/api/service/deployment/logs?deployment_id=${deploymentId}`;
		eventSource = new EventSource(url, { withCredentials: true });

		eventSource.addEventListener('log', (event: MessageEvent<string>) => {
			logs.push(event.data);
			streamState = 'connected';
		});

		eventSource.addEventListener('logs', (event: MessageEvent<string>) => {
			const prevLogs = JSON.parse(event.data) as string[];
			logs = [...logs, ...prevLogs];
			streamState = 'connected';
		});

		eventSource.onerror = () => closeStream();
	}

	$effect(() => {
		if (!open) {
			closeStream();
			return;
		}

		connectLogs();
		return closeStream;
	});
</script>

<Button variant="outline" size="sm" onclick={() => (open = true)}>View</Button>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed left-1/2 top-1/2 z-50 w-[95vw] max-w-3xl -translate-x-1/2 -translate-y-1/2 rounded-lg border bg-background p-4"
		>
			<Dialog.Title class="text-lg font-semibold">Deployment Logs</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground">
				{deploymentName}
			</Dialog.Description>

			<div class="mt-3 rounded-md border bg-muted/40 p-3">
				<p class="text-xs text-muted-foreground">state: {streamState}</p>

				<div
					class="mt-2 h-[50vh] overflow-auto rounded bg-background p-3 font-mono text-xs leading-5"
				>
					{#if logs.length === 0}
						<p class="text-muted-foreground">Waiting for deployment logs...</p>
					{:else}
						{#each logs as line, i (i)}
							<p>{line}</p>
						{/each}
					{/if}
				</div>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
