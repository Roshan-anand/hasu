<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import { useDeleteAppServiceMutation } from '@/features/services/mutation.svelte';
	import { Trash2 } from '@lucide/svelte';

	type Props = {
		serviceId: string;
		name: string;
	};

	let { serviceId, name }: Props = $props();
	let dialogOpen = $state(false);

	const deleteAppServiceMutation = useDeleteAppServiceMutation();

	function openDialog() {
		dialogOpen = true;
	}

	function closeDialog() {
		if (deleteAppServiceMutation.isPending) return;
		dialogOpen = false;
	}

	function deleteService() {
		if (deleteAppServiceMutation.isPending) return;
		deleteAppServiceMutation.mutate(
			{ service_id: serviceId },
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
			<Dialog.Title class="text-lg font-semibold">Delete App Service</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				This removes the runtime for <span class="font-medium text-foreground">{name}</span>.
			</Dialog.Description>

			<div class="flex justify-end gap-2 pt-5">
				<Button
					variant="outline"
					type="button"
					onclick={closeDialog}
					disabled={deleteAppServiceMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					variant="destructivesolid"
					type="button"
					onclick={deleteService}
					disabled={deleteAppServiceMutation.isPending}
				>
					{deleteAppServiceMutation.isPending ? 'Deleting...' : 'Delete'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
