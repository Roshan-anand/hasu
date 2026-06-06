<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
	import { Input } from '@/components/ui/input';
	import { Button } from '@/components/ui/button';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { useGetBranchDomainQuery } from '@/features/services/query.svelte';
	import { useUpdateBranchDomainMutation } from '@/features/services/mutation.svelte';

	let { serviceID }: { serviceID: string } = $props();

	const domainQuery = useGetBranchDomainQuery(() => serviceID);
	const updateDomainMutation = useUpdateBranchDomainMutation(() => serviceID);

	let domainInput = $state('');
	let portInput = $state('');

	$effect(() => {
		if (domainQuery.data) {
			domainInput = domainQuery.data.domain;
			portInput = domainQuery.data.port.toString();
		}
	});

	function submitDomain() {
		const domain = domainInput.trim();
		const port = Number(portInput);
		if (updateDomainMutation.isPending || domain === '' || Number.isNaN(port)) return;

		updateDomainMutation.mutate({
			service_id: serviceID,
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
		{#if domainQuery.isPending}
			<div class="flex flex-col gap-3">
				<Skeleton class="h-9 w-full" />
				<Skeleton class="h-9 w-full" />
			</div>
		{:else if domainQuery.isError}
			<p class="text-sm text-red-500">Failed to load domain details</p>
		{:else if domainQuery.data}
			<form
				class="flex flex-col gap-4"
				onsubmit={(e) => {
					e.preventDefault();
					submitDomain();
				}}
			>
				<div class="space-y-1.5">
					<Label for="domain-input">Domain</Label>
					<Input
						id="domain-input"
						placeholder="example.com"
						bind:value={domainInput}
						required
						disabled={updateDomainMutation.isPending}
					/>
				</div>

				<div class="space-y-1.5">
					<Label for="port-input">Port</Label>
					<Input
						id="port-input"
						placeholder="80"
						required
						type="number"
						bind:value={portInput}
						disabled={updateDomainMutation.isPending}
					/>
				</div>

				<div class="flex items-center justify-end">
					<Button type="submit" disabled={updateDomainMutation.isPending}>
						{updateDomainMutation.isPending ? 'Updating...' : 'Update'}
					</Button>
				</div>
			</form>
		{/if}
	</CardContent>
</Card>
