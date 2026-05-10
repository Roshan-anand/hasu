<script lang="ts">
	import { browser } from '$app/environment';

	let { branchId, open }: { branchId: string; open: boolean } = $props();

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
		if (!browser || branchId === '') return;

		closeStream();
		logs = [];
		streamState = 'connecting';

		const url = `/api/service/logs?branch_id=${branchId}`;
		eventSource = new EventSource(url, { withCredentials: true });

		eventSource.addEventListener('log', (event: MessageEvent<string>) => {
			logs.push(event.data);
			streamState = 'connected';
		});

		// eventSource.addEventListener('logs', (event: MessageEvent<string>) => {
		// 	const prevLogs = JSON.parse(event.data) as string[];
		// 	logs = [...logs, ...prevLogs];
		// 	streamState = 'connected';
		// });

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

<div class="mt-3 rounded-md border bg-muted/40 p-3">
	<p class="text-xs text-muted-foreground">ssstate: {streamState}</p>

	<div class="mt-2 h-[50vh] overflow-auto rounded bg-background p-3 font-mono text-xs leading-5">
		{#if logs.length === 0}
			<p class="text-muted-foreground">Waiting for server logs...</p>
		{:else}
			{#each logs as line, i (i)}
				<p>{line}</p>
			{/each}
		{/if}
	</div>
</div>
