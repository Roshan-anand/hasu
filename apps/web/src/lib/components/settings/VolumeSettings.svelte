<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { useRenameVolumeMutation } from '@/features/base';
	import { VolumeDeletion } from '@/components/conformation';
	import { Pencil, Trash2, Settings2 } from '@lucide/svelte';

	type Props = {
		volumeId: string;
		volumeName: string;
		displayName: string;
	};

	let { volumeId, volumeName, displayName }: Props = $props();

	const renameVolumeMutation = useRenameVolumeMutation();

	// Rename dialog state
	let renameDialogOpen = $state(false);
	let newDisplayName = $state('');

	// Delete dialog state (controlled via VolumeDeletion)
	let deleteDialogOpen = $state(false);

	function openRenameDialog() {
		newDisplayName = displayName || volumeName;
		renameDialogOpen = true;
	}

	function renameVolume() {
		if (newDisplayName.trim().length < 1 || renameVolumeMutation.isPending) return;
		renameVolumeMutation.mutate(
			{ id: volumeId, display_name: newDisplayName.trim() },
			{
				onSuccess: () => {
					renameDialogOpen = false;
				}
			}
		);
	}

	function openDeleteDialog() {
		deleteDialogOpen = true;
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="ghost" size="icon" class="z-50 size-7">
				<Settings2 class="size-3.5" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-40">
		<DropdownMenu.Item onSelect={openRenameDialog}>
			<Pencil class="size-4 mr-2" />
			Rename
		</DropdownMenu.Item>
		<DropdownMenu.Item onSelect={openDeleteDialog} class="text-destructive focus:text-destructive">
			<Trash2 class="size-4 mr-2" />
			Delete
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<!-- Rename Dialog -->
<Dialog.Root bind:open={renameDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Rename Volume</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Set a display label for <span class="font-medium text-foreground">{volumeName}</span>. This
				does not affect the underlying volume.
			</Dialog.Description>

			<form
				class="mt-4 space-y-4"
				onsubmit={(e) => {
					e.preventDefault();
					renameVolume();
				}}
			>
				<div class="space-y-1.5">
					<Label for="settings-rename-volume-name">Display Name</Label>
					<Input
						id="settings-rename-volume-name"
						placeholder="e.g. Production DB Backup"
						bind:value={newDisplayName}
						required
						minlength={1}
						disabled={renameVolumeMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={() => (renameDialogOpen = false)}
						disabled={renameVolumeMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={renameVolumeMutation.isPending}>
						{renameVolumeMutation.isPending ? 'Saving...' : 'Save'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<!-- Delete Dialog (reuses existing VolumeDeletion component) -->
<VolumeDeletion volume={volumeName} bind:dialogOpen={deleteDialogOpen} hideTrigger />
