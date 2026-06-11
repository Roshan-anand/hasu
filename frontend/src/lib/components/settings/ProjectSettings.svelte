<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { useRenameProjectMutation, getOrgState } from '@/features/base';
	import { ProjectDeletion } from '@/components/conformation';
	import { Pencil, Trash2, Settings2 } from '@lucide/svelte';

	type Props = {
		projectId: string;
		name: string;
	};

	let { projectId, name }: Props = $props();

	const renameProjectMutation = useRenameProjectMutation();
	const currentOrg = getOrgState();

	// Rename dialog state
	let renameDialogOpen = $state(false);
	let newName = $state('');

	// Delete dialog state (controlled via ProjectDeletion)
	let deleteDialogOpen = $state(false);

	function openRenameDialog() {
		newName = name;
		renameDialogOpen = true;
	}

	function renameProject() {
		if (newName.trim().length < 3 || renameProjectMutation.isPending) return;
		renameProjectMutation.mutate(
			{ project_id: projectId, org_id: currentOrg.id, name: newName.trim() },
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
			<Button {...props} variant="ghost" size="icon" class="z-50">
				<Settings2 class="size-4" />
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
			<Dialog.Title class="text-lg font-semibold">Rename Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Enter a new name for <span class="font-medium text-foreground">{name}</span>.
			</Dialog.Description>

			<form
				class="mt-4 space-y-4"
				onsubmit={(e) => {
					e.preventDefault();
					renameProject();
				}}
			>
				<div class="space-y-1.5">
					<Label for="settings-rename-project-name">Name</Label>
					<Input
						id="settings-rename-project-name"
						placeholder="New project name"
						bind:value={newName}
						required
						minlength={3}
						disabled={renameProjectMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={() => (renameDialogOpen = false)}
						disabled={renameProjectMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={renameProjectMutation.isPending}>
						{renameProjectMutation.isPending ? 'Saving...' : 'Save'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<!-- Delete Dialog (reuses existing ProjectDeletion component) -->
<ProjectDeletion {projectId} {name} bind:dialogOpen={deleteDialogOpen} hideTrigger />
