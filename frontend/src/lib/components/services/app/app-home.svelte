<script lang="ts">
	import { Card, CardContent } from '@/components/ui/card';
	import { Skeleton } from '@/components/ui/skeleton';
	import { useGetServiceDetailsQuery } from '@/features/services/query.svelte';
	import Icon from '@iconify/svelte';
	import * as Collapsible from '$lib/components/ui/collapsible/index.js';
	import AppLogs from './app-logs.svelte';
	import { ChevronRight, ChevronDown } from '@lucide/svelte';

	let { serviceId }: { serviceId: string } = $props();

	// query to fetch service details based on service type and id
	const serviceQuery = useGetServiceDetailsQuery(() => serviceId);
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
	{@const { name, branch_name, gh_repo_name, created_at, commit_msg, status, branch_id, domain } =
		serviceQuery.data}
	<Card class="flex-1 mb-5">
		<CardContent>
			<div class="flex flex-col gap-1 px-1">
				<h3 class="text-2xl font-bold flex items-center gap-1">
					<Icon icon="icon-park-outline:dot" class="text-green-500" />
					<span>
						{name}
					</span>
					<span class="bg-muted text-muted-foreground px-1 rounded-md">Production</span>
				</h3>
				<p class="flex items-center gap-2 mt-2 text-sm text-muted-foreground">
					<Icon icon="mingcute:git-branch-line" />
					<span>{gh_repo_name}</span>
					<span>{branch_name}</span>
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
					<AppLogs branchId={branch_id} {open} />
				</Collapsible.Content>
			</Collapsible.Root>
		</CardContent>
	</Card>
{:else}
	<p class="text-muted-foreground">Service not found</p>
{/if}
