<script lang="ts">
	import * as Dialog from '@/components/ui/dialog';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { useRenameOrgMutation } from '@/features/base';

	type Props = {
		open?: boolean;
		orgId?: string;
		orgName?: string;
	};

	let { open = $bindable(false), orgId = '', orgName = '' }: Props = $props();

	const renameOrgMutation = useRenameOrgMutation();

	let renameOrgName = $state('');

	$effect(() => {
		if (open) {
			renameOrgName = orgName;
		}
	});

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (renameOrgName.trim().length < 3 || renameOrgMutation.isPending) return;
		renameOrgMutation.mutate(
			{ org_id: orgId, name: renameOrgName.trim() },
			{
				onSuccess: () => {
					open = false;
				}
			}
		);
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Rename Organization</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Enter a new name for this organization.
			</Dialog.Description>

			<form class="mt-4 space-y-4" onsubmit={handleSubmit}>
				<div class="space-y-1.5">
					<Label for="settings-rename-org-name">Name</Label>
					<Input
						id="settings-rename-org-name"
						placeholder="New organization name"
						bind:value={renameOrgName}
						required
						minlength={3}
						disabled={renameOrgMutation.isPending}
					/>
				</div>

				<div class="flex justify-end gap-2 pt-1">
					<Button
						variant="outline"
						type="button"
						onclick={() => (open = false)}
						disabled={renameOrgMutation.isPending}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={renameOrgMutation.isPending}>
						{renameOrgMutation.isPending ? 'Saving...' : 'Save'}
					</Button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
