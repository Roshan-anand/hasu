<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
	import { Input } from '@/components/ui/input';
	import { Button } from '@/components/ui/button';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { useGetBranchDomainQuery } from '@/features/services/query.svelte';
	import { useUpdateBranchDomainMutation } from '@/features/services/mutation.svelte';

	let { serviceId }: { serviceId: string } = $props();

	const branchQuery = useGetBranchDomainQuery(() => serviceId);
	const updateDomainMutation = useUpdateBranchDomainMutation(() => serviceId);

	function submitDomain(branchId: string, domain: string, port: number) {
		if (updateDomainMutation.isPending || branchId === '' || domain === '' || Number.isNaN(port))
			return;

		updateDomainMutation.mutate({
			branch_id: branchId,
			domain,
			port
		});
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>Domain</CardTitle>
	</CardHeader>
	<CardContent>
		{#if branchQuery.isPending}
			<div class="flex flex-col gap-3">
				<Skeleton class="h-9 w-full" />
				<Skeleton class="h-9 w-full" />
			</div>
		{:else if branchQuery.isError}
			<p class="text-sm text-red-500">Failed to load domain details</p>
		{:else if !branchQuery.data || branchQuery.data.length === 0}
			<p class="text-sm text-muted-foreground">No branches found for this service</p>
		{:else}
			<div class="flex flex-col gap-4">
				{#each branchQuery.data as branch (branch.id)}
					{@const portId = `port-${branch.id}`}
					{@const domainId = `domain-${branch.id}`}
					<Card class="border-muted">
						<CardHeader>
							<CardTitle class="text-base">{branch.branch_name}</CardTitle>
						</CardHeader>
						<CardContent>
							<form
								class="flex flex-col gap-4"
								onsubmit={(e) => {
									e.preventDefault();
									const form = e.currentTarget as HTMLFormElement;
									const formData = new FormData(form);
									const domain = formData.get(domainId);
									const port = formData.get(portId);

									if (!domain || !port) return;
									const portnum = Number(port);

									submitDomain(branch.id, domain as string, portnum);
								}}
							>
								<div class="space-y-1.5">
									<Label for={domainId}>Domain</Label>
									<Input
										id={domainId}
										name={domainId}
										placeholder="example.com"
										defaultValue={branch.domain}
										required
										disabled={updateDomainMutation.isPending}
									/>
								</div>

								<div class="space-y-1.5">
									<Label for={portId}>Port</Label>
									<Input
										id={portId}
										name={portId}
										placeholder="80"
										required
										type="number"
										defaultValue={branch.port}
										disabled={updateDomainMutation.isPending}
									/>
								</div>

								<div class="flex items-center justify-end">
									<Button type="submit" disabled={updateDomainMutation.isPending}>
										{updateDomainMutation.isPending ? 'Updating...' : 'Update'}
									</Button>
								</div>
							</form>
						</CardContent>
					</Card>
				{/each}
			</div>
		{/if}
	</CardContent>
</Card>
