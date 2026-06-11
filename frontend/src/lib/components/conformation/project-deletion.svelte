<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import { useDeleteProjectMutation } from '@/features/base';
	import { Trash2 } from '@lucide/svelte';

	type Props = {
		projectId: string;
		name: string;
		dialogOpen?: boolean;
		hideTrigger?: boolean;
	};

	let { projectId, name, dialogOpen = $bindable(false), hideTrigger = false }: Props = $props();
	let deleteVolumes = $state<string[]>([]);

	const deleteProjectMutation = useDeleteProjectMutation();

	function openDialog() {
		dialogOpen = true;
		deleteVolumes = [];
	}

	function closeDialog() {
		if (deleteProjectMutation.isPending) return;
		dialogOpen = false;
	}

	function deleteProject() {
		if (deleteProjectMutation.isPending) return;
		deleteProjectMutation.mutate(
			{ project_id: projectId, volumes: deleteVolumes },
			{
				onSuccess: () => {
					dialogOpen = false;
				}
			}
		);
	}
</script>

{#if !hideTrigger}
	<Button
		variant="destructive"
		size="sm"
		class="z-50 absolute top-1/2 right-0 -translate-y-1/2"
		onclick={openDialog}
	>
		<Trash2 />
		Delete
	</Button>
{/if}

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Delete Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Delete <span class="font-medium text-foreground">{name}</span> and optionally remove preserved
				data volumes.
			</Dialog.Description>

			<div class="flex justify-end gap-2 pt-5">
				<Button
					variant="outline"
					type="button"
					onclick={closeDialog}
					disabled={deleteProjectMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					variant="destructivesolid"
					type="button"
					onclick={deleteProject}
					disabled={deleteProjectMutation.isPending}
				>
					{deleteProjectMutation.isPending ? 'Deleting...' : 'Delete'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
