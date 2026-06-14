<script lang="ts">
	import * as Dialog from '@/components/ui/dialog';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { useCreateOrgMutation } from '@/features/base';

	type Props = {
		open?: boolean;
	};

	let { open = $bindable(false) }: Props = $props();

	const createOrgMutation = useCreateOrgMutation();

	let createOrgName = $state('');

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (createOrgName.trim().length < 3 || createOrgMutation.isPending) return;
		createOrgMutation.mutate(
			{ name: createOrgName.trim() },
			{
				onSuccess: () => {
					createOrgName = '';
					open = false;
				}
			}
		);
	}

	function handleCancel() {
		if (createOrgMutation.isPending) return;
		createOrgName = '';
		open = false;
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Create Organization</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Add a name for your organization.
			</Dialog.Description>

			<form class="mt-4 space-y-4" onsubmit={handleSubmit}>
				<div class="space-y-1.5">
					<Label for="settings-create-org-name">Name</Label>
					<Input
						id="settings-create-org-name"
						placeholder="Organization name"
						bind:value={createOrgName}
						required
						minlength={3}
						disabled={createOrgMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={handleCancel}
						disabled={createOrgMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={createOrgMutation.isPending}>
						{createOrgMutation.isPending ? 'Creating...' : 'Create'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
