<script lang="ts">
	import { Card, CardContent } from '@/components/ui/card';
	import { Skeleton } from '@/components/ui/skeleton';
	import Icon from '@iconify/svelte';
	import * as Collapsible from '$lib/components/ui/collapsible/index.js';
	import AppLogs from './app-logs.svelte';
	import { ChevronRight, ChevronDown } from '@lucide/svelte';
	import { Button } from '@/components/ui/button';
	import {
		useRebuildServiceMutation,
		useRollbackServiceMutation
	} from '@/features/deployments/mutation.svelte';
	import { useGetAppServiceDetailsQuery } from '@/features/services/query.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import type { ServiceType } from '@/types';

	let { serviceID, project }: { serviceID: string; project: string } = $props();

	const serviceQuery = useGetAppServiceDetailsQuery(() => serviceID);
	const rebuildService = useRebuildServiceMutation();
	const rollBackService = useRollbackServiceMutation();

	let open = $state(false);
</script>

{#if serviceQuery.isPending}
	<div class="flex flex-col gap-4">
		<Skeleton class="h-[50vh] w-full" />
		<Skeleton class="h-[15vh] w-full" />
	</div>
{:else if serviceQuery.isError}
	<p class="text-red-500">Failed to load service details</p>
{:else if serviceQuery.data}
	{@const { name, branch, gh_repo_name, created_at, commit_msg, status, domain } =
		serviceQuery.data}
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

					<div>
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
												resolve('/(protected)/[project]/[service_type]/[service]?tab=deployment', {
													service_type: 'app' as ServiceType,
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
				</p>
				<p>{commit_msg}</p>
				<section class="flex items-center gap-4">
					<div>
						status : {status}
					</div>
					<div>created at : {created_at}</div>
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
