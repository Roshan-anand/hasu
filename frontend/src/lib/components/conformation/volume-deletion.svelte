<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import { useDeleteVolumeMutation } from '@/features/base';
	import { Trash2 } from '@lucide/svelte';

	type Props = {
		volume: string;
		label?: string;
		onDeleted?: () => void;
		dialogOpen?: boolean;
		hideTrigger?: boolean;
	};

	let {
		volume,
		label = 'Delete',
		onDeleted,
		dialogOpen = $bindable(false),
		hideTrigger = false
	}: Props = $props();

	const deleteVolumeMutation = useDeleteVolumeMutation();

	function openDialog() {
		dialogOpen = true;
	}

	function closeDialog() {
		if (deleteVolumeMutation.isPending) return;
		dialogOpen = false;
	}

	function deleteVolume() {
		if (deleteVolumeMutation.isPending) return;
		const done = () => {
			dialogOpen = false;
			onDeleted?.();
		};
		deleteVolumeMutation.mutate({ volumes: [volume] }, { onSuccess: done });
	}
</script>

{#if !hideTrigger}
	<Button variant="destructive" size="sm" onclick={openDialog}>
		<Trash2 />
		{label}
	</Button>
{/if}

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Delete Orphan Volume</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				This permanently removes the data volume.
			</Dialog.Description>

			<div class="mt-2 rounded-md border bg-muted/40 p-3 text-xs text-muted-foreground">
				Volume: {volume}
			</div>

			<div class="flex justify-end gap-2 pt-5">
				<Button
					variant="outline"
					type="button"
					onclick={closeDialog}
					disabled={deleteVolumeMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					variant="destructivesolid"
					type="button"
					onclick={deleteVolume}
					disabled={deleteVolumeMutation.isPending}
				>
					{deleteVolumeMutation.isPending ? 'Deleting...' : 'Delete'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
