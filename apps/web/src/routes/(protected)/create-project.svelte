<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { useCreateProjectMutation } from '@/features/base';

	let dialogOpen = $state(false);
	let projectName = $state('');
	const createProjectMutation = useCreateProjectMutation();
	const canCreateProject = $derived.by(() => projectName.trim().length >= 3);

	function createProject() {
		if (!canCreateProject || createProjectMutation.isPending) return;
		createProjectMutation.mutate(
			{ name: projectName.trim() },
			{
				onSuccess: () => {
					projectName = '';
					dialogOpen = false;
				}
			}
		);
	}

	function openDialog() {
		dialogOpen = true;
	}

	function closeDialog() {
		if (createProjectMutation.isPending) return;
		dialogOpen = false;
	}
</script>

<Button onclick={openDialog} class="h-full">Create</Button>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Create Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground">
				Add a name for your project.
			</Dialog.Description>

			<form
				class="mt-4 space-y-4"
				onsubmit={(e) => {
					e.preventDefault();
					createProject();
				}}
			>
				<div class="space-y-1.5">
					<Label class="my-1" for="create-project-name">Name</Label>
					<Input
						id="create-project-name"
						placeholder="Project name"
						bind:value={projectName}
						required
						minlength={3}
						disabled={createProjectMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={closeDialog}
						disabled={createProjectMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={!canCreateProject || createProjectMutation.isPending}>
						{createProjectMutation.isPending ? 'Creating...' : 'Create'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
