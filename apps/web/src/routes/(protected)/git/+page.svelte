<script lang="ts">
	import { DotmSquare } from '@/components/loader';
	import { Button } from '@/components/ui/button';
	import { useDeleteGithubAppMutation } from '@/features/git';
	import { useGithubAppsQuery } from '@/features/git';
	import { GitProvidersList } from '@/features/services';
	import Icon from '@iconify/svelte';

	const getGithubAppsQuery = useGithubAppsQuery();
	const deleteGithubAppMutation = useDeleteGithubAppMutation();

	const providerRedirect = (loc: string) => (window.location.href = loc);

	const formatCreatedAt = (createdAt: string) => {
		const parsedDate = new Date(createdAt);
		if (Number.isNaN(parsedDate.getTime())) return createdAt;
		return parsedDate.toLocaleString();
	};
</script>

<section class="p-2">
	<h1 class="my-2">Connect any git provider</h1>

	<section class="flex items-center gap-4 w-full">
		{#each Array.from(GitProvidersList) as [key, p] (key)}
			<Button
				id={p.name}
				variant="outline"
				disabled={p.createApi == ''}
				onclick={() => providerRedirect(p.createApi)}
				class="flex-1"
			>
				<Icon icon={p.icon} width="24" height="24" />
				<p>{p.name}</p>
			</Button>
		{/each}
	</section>
</section>

<hr class="my-3" />

<section class="flex-1">
	{#if getGithubAppsQuery.isPending}
		<div class="size-full flex items-center justify-center">
			<DotmSquare size={65} dotSize={8} />
		</div>
	{:else if getGithubAppsQuery.isError}
		<p class="text-destructive">Failed to load provider details.</p>
	{:else if !getGithubAppsQuery.data || getGithubAppsQuery.data.length === 0}
		<div class="flex items-center gap-2 text-muted-foreground size-full justify-center">
			<Icon icon="material-icon-theme:git" width="24" height="24" />
			<p>No provider connected</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each getGithubAppsQuery.data as app (app.app_id)}
				<div class="rounded-lg border bg-card text-card-foreground shadow-sm p-4 space-y-3">
					<div class="flex items-start justify-between gap-3">
						<div class="flex items-center gap-2">
							<Icon icon="meteor-icons:github" width="24" height="24" />
							<div>
								<h2 class="font-semibold">GitHub</h2>
								<p class="text-sm text-muted-foreground">{app.name}</p>
							</div>
						</div>

						<Button
							variant="destructive"
							size="sm"
							onclick={() => deleteGithubAppMutation.mutate({ app_id: app.app_id })}
							disabled={deleteGithubAppMutation.isPending}
						>
							{deleteGithubAppMutation.isPending ? 'Deleting...' : 'Delete'}
						</Button>
					</div>

					<div class="text-sm text-muted-foreground space-y-1">
						<span class="font-medium text-foreground">Created:</span>
						{formatCreatedAt(app.created_at)}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>
