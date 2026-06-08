<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import { Input } from '@/components/ui/input';
	import { Search, X, GitPullRequest } from '@lucide/svelte';
	import { useGetGithubPRListQuery } from '@/features/services';
	import type { PRInfo } from '@/features/services';
	import { DotmSquare } from '@/components/loader';

	let { serviceId, onSelect }: { serviceId: string; onSelect: (pr: PRInfo) => void } = $props();

	let dialogOpen = $state(false);
	let searchQuery = $state('');
	let selectedPRState = $state<PRInfo | null>(null);

	const prQuery = useGetGithubPRListQuery(() => serviceId);

	const handleOpenPreviewDialog = () => {
		dialogOpen = true;
		searchQuery = '';
		selectedPRState = null;
		prQuery.refetch();
	};

	const handleSelectPR = (pr: PRInfo) => {
		selectedPRState = pr;
	};

	const handleConfirm = () => {
		if (selectedPRState) {
			onSelect(selectedPRState);
		}
		dialogOpen = false;
	};

	const filteredPRs = $derived.by(() => {
		if (!prQuery.data) return [];
		const q = searchQuery.toLowerCase().trim();
		if (!q) return prQuery.data;

		return prQuery.data.filter(
			(pr) => pr.title.toLowerCase().includes(q) || pr.number.toString().includes(q)
		);
	});
</script>

<Button variant="outline" onclick={handleOpenPreviewDialog}>Create Preview</Button>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg flex flex-col max-h-[85vh]"
		>
			<Dialog.Title class="text-lg font-semibold">Create Preview</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground">
				Select a pull request to deploy as a preview instance.
			</Dialog.Description>

			<div class="relative my-3">
				<Input
					id="pr-search-single"
					placeholder="Search PRs by title or number..."
					class="pl-9 pr-4 py-2"
					bind:value={searchQuery}
				/>
				<Search class="absolute left-3 top-3 size-4 text-muted-foreground" />
			</div>

			<div
				class="flex-1 overflow-y-auto min-h-[200px] max-h-[40vh] border rounded-lg p-2 bg-muted/30"
			>
				{#if prQuery.isPending}
					<div class="size-full flex items-center justify-center py-8">
						<DotmSquare size={40} dotSize={5} />
					</div>
				{:else if prQuery.isError}
					<p class="text-sm text-red-500 text-center py-8">Failed to load pull requests</p>
				{:else if filteredPRs.length > 0}
					<div class="space-y-1">
						{#each filteredPRs as pr (pr.id)}
							{@const isSelected = selectedPRState?.id === pr.id}
							<button
								type="button"
								onclick={() => handleSelectPR(pr)}
								class="w-full text-left px-3 py-2 rounded-md text-sm transition-all flex items-start gap-2.5 {isSelected
									? 'bg-primary text-primary-foreground font-medium shadow-sm'
									: 'hover:bg-accent text-foreground'}"
							>
								<GitPullRequest class="size-4 shrink-0 mt-0.5" />
								<div class="flex-1 min-w-0">
									<p class="truncate">{pr.title}</p>
									<p
										class="text-xs {isSelected
											? 'text-primary-foreground/80'
											: 'text-muted-foreground'}"
									>
										#{pr.number} • {pr.state}
									</p>
								</div>
							</button>
						{/each}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground text-center py-8">No open pull requests found</p>
				{/if}
			</div>

			{#if selectedPRState}
				<div
					class="mt-3 p-2.5 bg-accent/60 rounded-lg text-sm border flex items-center justify-between"
				>
					<div class="flex items-center gap-2 min-w-0">
						<GitPullRequest class="size-4 text-primary shrink-0" />
						<span class="truncate font-medium">
							Selected: #{selectedPRState.number} - {selectedPRState.title}
						</span>
					</div>
					<Button
						variant="ghost"
						size="icon"
						class="h-6 w-6 shrink-0"
						onclick={() => (selectedPRState = null)}
					>
						<X class="h-4 w-4" />
					</Button>
				</div>
			{/if}

			<div class="flex justify-end gap-2 pt-4 border-t mt-4">
				<Button variant="outline" type="button" onclick={() => (dialogOpen = false)}>Cancel</Button>
				<Button type="button" disabled={!selectedPRState} onclick={handleConfirm}>Create</Button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
