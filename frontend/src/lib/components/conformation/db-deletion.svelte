<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Checkbox } from '@/components/ui/checkbox';
	import * as Dialog from '@/components/ui/dialog';
	import { Label } from '@/components/ui/label';
	import { useDeletePsqlServiceMutation } from '@/features/services/mutation.svelte';
	import { Trash2 } from '@lucide/svelte';

	type Props = {
		serviceId: string;
		name: string;
	};

	let { serviceId, name }: Props = $props();
	let dialogOpen = $state(false);
	let keepData = $state(true);

	const deletePsqlServiceMutation = useDeletePsqlServiceMutation();

	function openDialog() {
		dialogOpen = true;
	}

	function closeDialog() {
		if (deletePsqlServiceMutation.isPending) return;
		dialogOpen = false;
	}

	function deleteService() {
		if (deletePsqlServiceMutation.isPending) return;
		deletePsqlServiceMutation.mutate(
			{
				service_id: serviceId,
				keep_data: keepData
			},
			{
				onSuccess: () => {
					dialogOpen = false;
				}
			}
		);
	}
</script>

<Button variant="destructive" size="sm" class="z-20" onclick={openDialog}>
	<Trash2 />
	Delete
</Button>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Delete Database Service</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Delete <span class="font-medium text-foreground">{name}</span> and choose whether to keep its
				data as an orphan volume.
			</Dialog.Description>

			<div class="mt-4 flex items-center space-x-2 rounded-md border p-3">
				<Checkbox id="keep-data" bind:checked={keepData} />
				<Label
					for="keep-data"
					class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
				>
					Keep data
				</Label>
			</div>

			<div class="flex justify-end gap-2 pt-5">
				<Button
					variant="outline"
					type="button"
					onclick={closeDialog}
					disabled={deletePsqlServiceMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					variant="destructivesolid"
					type="button"
					onclick={deleteService}
					disabled={deletePsqlServiceMutation.isPending}
				>
					{deletePsqlServiceMutation.isPending ? 'Deleting...' : 'Delete'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
