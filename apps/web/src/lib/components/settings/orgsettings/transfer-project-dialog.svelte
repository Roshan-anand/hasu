<script lang="ts">
	import * as Dialog from '@/components/ui/dialog';
	import { Button } from '@/components/ui/button';
	import { Label } from '@/components/ui/label';
	import { Check, Building2 } from '@lucide/svelte';
	import { useTransferProjectMutation } from '@/features/base';
	import { toast } from 'svelte-sonner';

	type Org = {
		id: string;
		name: string;
	};

	type Props = {
		project?: { id: string; name: string } | null;
		orgs?: Org[];
		sourceOrgId?: string;
	};

	let { project = null, orgs = [], sourceOrgId = '' }: Props = $props();

	const transferProjectMutation = useTransferProjectMutation();

	let open = $state(false);
	const targetOrgs = $derived(orgs.filter((o) => o.id !== sourceOrgId));
	let selectedOrgId = $state('');

	$effect(() => {
		if (project) {
			open = true;
			selectedOrgId = '';
		}
	});

	function handleClose() {
		open = false;
	}

	function handleConfirm() {
		if (!selectedOrgId || transferProjectMutation.isPending || !project) return;
		transferProjectMutation.mutate(
			{ project_id: project.id, target_org_id: selectedOrgId },
			{
				onSuccess: () => {
					open = false;
					toast.success(`Project "${project!.name}" transferred`);
				}
			}
		);
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-50 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Transfer Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground mt-1">
				Move <strong>{project?.name ?? ''}</strong> to another organization instead of deleting it.
			</Dialog.Description>

			<div class="mt-4 space-y-3">
				<Label for="transfer-target-org">Target Organization</Label>
				{#if targetOrgs.length === 0}
					<p class="text-sm text-muted-foreground">
						No other organizations available. Create one first.
					</p>
				{:else}
					<div class="space-y-2">
						{#each targetOrgs as org (org.id)}
							<button
								type="button"
								onclick={() => (selectedOrgId = org.id)}
								class="flex w-full items-center gap-3 rounded-lg border px-3 py-2 text-left text-sm transition-colors hover:bg-accent
									{selectedOrgId === org.id ? 'border-primary ring-2 ring-primary/20' : 'border-border'}"
							>
								<div class="flex size-8 items-center justify-center rounded-md bg-primary/10">
									<Building2 class="size-4 text-primary" />
								</div>
								<span class="flex-1 font-medium">{org.name}</span>
								{#if selectedOrgId === org.id}
									<Check class="size-4 text-primary" />
								{/if}
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<div class="flex justify-end gap-2 mt-6 pt-2 border-t">
				<Button
					variant="outline"
					type="button"
					onclick={handleClose}
					disabled={transferProjectMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					onclick={handleConfirm}
					disabled={!selectedOrgId || transferProjectMutation.isPending}
				>
					{transferProjectMutation.isPending ? 'Transferring...' : 'Transfer'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
