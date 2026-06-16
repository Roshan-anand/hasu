<script lang="ts">
	import { Card, CardContent } from '@/components/ui/card';
	import { Skeleton } from '@/components/ui/skeleton';
	import Icon from '@iconify/svelte';
	import * as Collapsible from '$lib/components/ui/collapsible/index.js';
	import AppLogs from './app-logs.svelte';
	import { ChevronRight, ChevronDown } from '@lucide/svelte';
	import { Button } from '@/components/ui/button';
	import { useRebuildServiceMutation, useRollbackServiceMutation } from '@/features/deployments';
	import {
		useGetAppServiceDetailsQuery,
		usePauseAppServiceMutation,
		useResumeAppServiceMutation
	} from '@/features/services';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import AppServicePRPreviewButton from './AppServicePRPreviewButton.svelte';
	import { X } from '@lucide/svelte';
	import type { PRInfo } from '@/features/services';

	let { serviceID, project }: { serviceID: string; project: string } = $props();

	const copyToClipboard = async (text: string, label: string) => {
		try {
			await navigator.clipboard.writeText(text);
			toast.success(`${label} copied to clipboard`);
		} catch {
			toast.error('Failed to copy to clipboard');
		}
	};

	const serviceQuery = useGetAppServiceDetailsQuery(() => serviceID);
	const rebuildService = useRebuildServiceMutation();
	const rollBackService = useRollbackServiceMutation();
	const pauseService = usePauseAppServiceMutation(() => serviceID);
	const resumeService = useResumeAppServiceMutation(() => serviceID);

	let open = $state(false);
	let selectedPR = $state<PRInfo | null>(null);
</script>

{#if serviceQuery.isPending}
	<div class="flex flex-col gap-4">
		<Skeleton class="h-[50vh] w-full" />
		<Skeleton class="h-[15vh] w-full" />
	</div>
{:else if serviceQuery.isError}
	<p class="text-red-500">Failed to load service details</p>
{:else if serviceQuery.data}
	{@const {
		name,
		branch,
		gh_repo_name,
		created_at,
		commit_msg,
		status,
		replicas,
		domain,
		internal_url,
		port,
		is_public
	} = serviceQuery.data}
	{#if selectedPR}
		<div
			class="mb-4 p-2 bg-accent text-accent-foreground rounded-lg flex items-center justify-between border"
		>
			<div class="flex items-center gap-2">
				<span class="text-xs bg-primary text-primary-foreground px-2 py-0.5 rounded font-semibold"
					>PR Preview</span
				>
				<span class="font-medium text-sm">#{selectedPR.number}: {selectedPR.title}</span>
			</div>
			<Button variant="ghost" size="icon" class="h-6 w-6" onclick={() => (selectedPR = null)}>
				<X class="h-4 w-4" />
			</Button>
		</div>
	{/if}
	<Card class="flex-1 mb-5">
		<CardContent>
			<div class="flex flex-col gap-1 px-1">
				<h3 class="font-bold flex justify-between items-center">
					<div class="flex items-center gap-2 text-2xl">
						<Icon icon="icon-park-outline:dot" class="text-green-500" />
						<span>
							{name}
						</span>
						<span class="bg-muted text-muted-foreground px-1 rounded-md">Production</span>
					</div>

					<div class="flex items-center gap-2">
						<AppServicePRPreviewButton serviceId={serviceID} onSelect={(pr) => (selectedPR = pr)} />
						{#if status === 'paused'}
							<Button disabled={resumeService.isPending} onclick={() => resumeService.mutate()}>
								Resume
							</Button>
						{:else}
							<Button
								variant="secondary"
								disabled={pauseService.isPending || status !== 'ready'}
								onclick={() => pauseService.mutate()}
							>
								Pause
							</Button>
						{/if}
						<Button
							disabled={rebuildService.isPending}
							onclick={() =>
								rebuildService.mutate(
									{
										service_id: serviceID
									},
									{
										onSuccess: ({ data }) => {
											goto(
												resolve('/(protected)/[project]/[service]?tab=deployment', {
													project,
													service: data
												})
											);
										}
									}
								)}>Redeploy</Button
						>
						<Button
							onclick={() =>
								rollBackService.mutate({
									service_id: serviceID
								})}
							disabled={rollBackService.isPending}>Rollback</Button
						>
					</div>
				</h3>
				<p class="flex items-center gap-2 mt-2 text-sm text-muted-foreground">
					<Icon icon="mingcute:git-branch-line" />
					<span>{gh_repo_name}</span>
					<span>{branch}</span>
					{#if is_public}
						{#if domain === ''}
							<span>No domain specified</span>
						{:else}
							<!--  eslint-disable svelte/no-navigation-without-resolve  -->
							<a
								href={domain.includes('https://') ? domain : `https://${domain}`}
								target="_blank"
								class="flex items-center gap-1 underline underline-offset-2 hover:text-primary"
							>
								<span>{domain}</span>
								<Icon icon="lucide:external-link" class="w-4 h-4" />
							</a>
						{/if}
					{:else}
						<span class="text-muted-foreground">Internal service</span>
					{/if}
				</p>
				<p>{commit_msg}</p>
				<section class="flex items-center gap-4">
					<div>
						deployment :
						{#if status === 'paused'}
							<span class="text-yellow-500 font-medium">paused</span>
						{:else}
							{status}
						{/if}
					</div>
					<div>replicas : {replicas}</div>
					<div>created at : {created_at}</div>
				</section>
				<section class="mt-2 rounded-md border bg-muted/30 p-3">
					<h4 class="mb-1 text-xs font-medium text-muted-foreground">Internal URL</h4>
					<div class="flex items-center gap-2 text-sm">
						<Icon icon="lucide:network" class="h-4 w-4 text-muted-foreground" />
						<code class="rounded bg-muted px-2 py-0.5 font-mono text-xs">
							{internal_url}
						</code>
						<span class="text-xs text-muted-foreground">(port {port})</span>
						<button
							type="button"
							onclick={() => copyToClipboard(internal_url, 'Internal URL')}
							class="ml-auto rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
							title="Copy internal URL"
						>
							<Icon icon="lucide:copy" class="h-4 w-4" />
						</button>
					</div>
				</section>
			</div>
		</CardContent>
	</Card>
	<Card>
		<CardContent>
			<Collapsible.Root bind:open>
				<Collapsible.Trigger class="flex items-center gap-2 w-full">
					{#if open}
						<ChevronDown class="h-4 w-4" />
					{:else}
						<ChevronRight class="h-4 w-4" />
					{/if}
					<span> application logs </span>
				</Collapsible.Trigger>
				<Collapsible.Content>
					<AppLogs {serviceID} {open} />
				</Collapsible.Content>
			</Collapsible.Root>
		</CardContent>
	</Card>
{:else}
	<p class="text-muted-foreground">Service not found</p>
{/if}
