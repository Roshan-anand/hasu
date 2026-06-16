<script lang="ts">
	import { browser } from '$app/environment';
	import { Terminal } from '@xterm/xterm';
	import { FitAddon } from '@xterm/addon-fit';
	import '@xterm/xterm/css/xterm.css';

	let { serviceID, open }: { serviceID: string; open: boolean } = $props();

	let terminalEl = $state<HTMLDivElement>();
	let streamState = $state<'connecting' | 'connected' | 'closed'>('closed');

	let term: Terminal | null = null;
	let fitAddon: FitAddon | null = null;
	let eventSource: EventSource | null = null;

	function teardownTerminal() {
		fitAddon?.dispose();
		fitAddon = null;
		term?.dispose();
		term = null;
	}

	function closeStream() {
		if (!eventSource) return;
		eventSource.close();
		eventSource = null;
	}

	function write(msg: string) {
		term?.writeln(msg);
	}

	function connectLogs() {
		if (!browser || serviceID === '' || !term) return;
		closeStream();
		streamState = 'connecting';
		term.clear();
		term.writeln('\x1b[33m●\x1b[0m Connecting to service logs...');

		const url = `/api/service/predef/logs?service_id=${serviceID}`;
		eventSource = new EventSource(url, { withCredentials: true });

		eventSource.addEventListener('log', (event: MessageEvent<string>) => {
			if (streamState !== 'connected') {
				streamState = 'connected';
				term!.clear();
			}
			write(event.data);
		});

		eventSource.onerror = () => {
			closeStream();
			streamState = 'closed';
		};
	}

	$effect(() => {
		if (!terminalEl || !open) return;

		// init xterm
		term = new Terminal({
			fontSize: 13,
			fontFamily:
				"'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, 'Courier New', monospace",
			disableStdin: true,
			cursorBlink: false,
			convertEol: true,
			scrollback: 5000,
			theme: {
				background: '#0a0a0a',
				foreground: '#fff',
				cursor: '#ffffff',
				cursorAccent: '#000000',
				selectionBackground: '#444444',
				selectionForeground: '#ffffff'
			}
		});

		fitAddon = new FitAddon();
		term.loadAddon(fitAddon);
		term.open(terminalEl);
		term.writeln('\x1b[2mWaiting for service logs...\x1b[0m');

		const onResize = () => fitAddon?.fit();
		window.addEventListener('resize', onResize);
		requestAnimationFrame(() => fitAddon?.fit());

		// start streaming
		connectLogs();

		return () => {
			window.removeEventListener('resize', onResize);
			closeStream();
			streamState = 'closed';
			teardownTerminal();
		};
	});
</script>

<section class="mt-3">
	<div class="rounded-md border bg-muted/40 p-3">
		<div class="flex items-center justify-between mb-2">
			<h3 class="text-sm font-medium">Service Logs</h3>
			<span
				class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium
					{streamState === 'connected'
					? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
					: streamState === 'connecting'
						? 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
						: 'bg-muted text-muted-foreground'}"
			>
				<span
					class="h-1.5 w-1.5 rounded-full
						{streamState === 'connected'
						? 'bg-emerald-500 animate-pulse'
						: streamState === 'connecting'
							? 'bg-amber-500 animate-pulse'
							: 'bg-muted-foreground'}"
				></span>
				{streamState}
			</span>
		</div>

		<div class="h-[50vh] overflow-hidden rounded bg-background">
			<div bind:this={terminalEl} class="h-full w-full border-2"></div>
		</div>
	</div>
</section>
