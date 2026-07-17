<script lang="ts">
	import { useScaleAppServiceMutation } from '@/features/services';
	import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
	import { Button } from '@/components/ui/button';
	import * as Select from '@/components/ui/select';

	let {
		serviceID,
		replicas = $bindable(1)
	}: {
		serviceID: string;
		replicas?: number;
	} = $props();

	const scaleMutation = useScaleAppServiceMutation(() => serviceID);

	const replicaOptions = [1, 2, 3, 4, 5];

	let initialReplicas = $state(replicas);
	let selectedReplicas = $state(replicas);

	$effect(() => {
		initialReplicas = replicas;
		selectedReplicas = replicas;
	});

	const isDirty = $derived(selectedReplicas !== initialReplicas);

	function handleReplicaChange(value: string) {
		selectedReplicas = Number(value);
	}

	function handleApply() {
		if (!isDirty || scaleMutation.isPending) return;
		scaleMutation.mutate({
			service_id: serviceID,
			replicas: selectedReplicas
		});
	}
</script>

<Card>
	<CardHeader>
		<CardTitle class="text-lg">Replicas</CardTitle>
	</CardHeader>
	<CardContent>
		<div class="space-y-1.5">
			<p class="text-sm text-muted-foreground">Number of running instances for this service.</p>
			<div class="flex items-end gap-2">
				<div class="flex-1">
					<Select.Root
						type="single"
						value={String(selectedReplicas)}
						onValueChange={handleReplicaChange}
						disabled={scaleMutation.isPending}
					>
						<Select.Trigger class="w-full" id="replica-select">
							{selectedReplicas}
							{selectedReplicas === 1 ? 'replica' : 'replicas'}
						</Select.Trigger>
						<Select.Content>
							{#each replicaOptions as count (count)}
								<Select.Item
									value={String(count)}
									label={`${count} ${count === 1 ? 'replica' : 'replicas'}`}
								/>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
				<Button disabled={!isDirty || scaleMutation.isPending} onclick={handleApply}>
					{scaleMutation.isPending ? 'Applying...' : 'Apply'}
				</Button>
			</div>
		</div>
	</CardContent>
</Card>
