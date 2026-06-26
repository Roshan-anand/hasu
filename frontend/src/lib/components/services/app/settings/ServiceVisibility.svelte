<script lang="ts">
	import { useUpdateServiceDomainMutation } from '@/features/services';
	import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Switch } from 'bits-ui';
	import { cn } from '@/utils';

	let {
		serviceID,
		initialDomain = $bindable(''),
		initialPort = $bindable(0),
		initialIsPublic = $bindable(false)
	}: {
		serviceID: string;
		initialDomain?: string;
		initialPort?: number;
		initialIsPublic?: boolean;
	} = $props();

	const updateDomainMutation = useUpdateServiceDomainMutation(() => serviceID);

	let isPublic = $state(initialIsPublic);
	let domainInput = $state(initialDomain);
	let portInput = $state(String(initialPort));
	let domainError = $state('');
	let portError = $state('');

	$effect(() => {
		isPublic = initialIsPublic;
		domainInput = initialDomain;
		portInput = String(initialPort);
	});

	const isDirty = $derived(
		isPublic !== initialIsPublic ||
			domainInput !== initialDomain ||
			portInput !== String(initialPort)
	);

	function handleTogglePublic(checked: boolean) {
		isPublic = checked;
		domainError = '';
		portError = '';
	}

	function handleSaveDomain() {
		if (updateDomainMutation.isPending) return;

		domainError = '';
		portError = '';

		const port = Number(portInput);
		if (Number.isNaN(port) || port < 1 || port > 65535) {
			portError = 'Port must be a number between 1 and 65535';
			return;
		}

		const domain = isPublic ? domainInput.trim() : '';
		if (isPublic && domain === '') {
			domainError = 'Domain is required when the service is public';
			return;
		}

		updateDomainMutation.mutate({
			service_id: serviceID,
			domain,
			port,
			is_public: isPublic
		});
	}
</script>

<Card>
	<CardHeader>
		<CardTitle class="text-lg">Service Visibility</CardTitle>
	</CardHeader>
	<CardContent>
		<div class="flex items-center justify-between">
			<div class="space-y-0.5">
				<Label for="visibility-switch">Public Service</Label>
				<p class="text-sm text-muted-foreground">
					Make this service accessible via a public domain
				</p>
			</div>
			<Switch.Root
				id="visibility-switch"
				checked={isPublic}
				onCheckedChange={handleTogglePublic}
				disabled={updateDomainMutation.isPending}
				class={cn(
					'relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors',
					'focus-visible:ring-ring/50 focus-visible:ring-3 focus-visible:outline-none',
					isPublic ? 'bg-primary' : 'bg-input',
					'disabled:cursor-not-allowed disabled:opacity-50'
				)}
			>
				<Switch.Thumb
					class={cn(
						'pointer-events-none block size-5 rounded-full bg-background shadow-sm ring-0 transition-transform',
						isPublic ? 'translate-x-5' : 'translate-x-0'
					)}
				/>
			</Switch.Root>
		</div>

		<form
			class="flex flex-col gap-4 mt-5 border-t pt-5"
			onsubmit={(e) => {
				e.preventDefault();
				handleSaveDomain();
			}}
		>
			{#if isPublic}
				<div class="space-y-1.5">
					<Label for="settings-domain">Domain</Label>
					<Input
						id="settings-domain"
						placeholder="example.com"
						bind:value={domainInput}
						required={isPublic}
						type="text"
						class={cn(domainError && 'border-destructive focus-visible:ring-destructive')}
						oninput={() => {
							if (domainError) domainError = '';
						}}
						disabled={updateDomainMutation.isPending}
					/>
					{#if domainError}
						<p class="text-sm text-destructive">{domainError}</p>
					{/if}
				</div>

				<div class="space-y-1.5">
					<Label for="settings-port">Port</Label>
					<Input
						id="settings-port"
						placeholder="80"
						required
						type="number"
						min="1"
						max="65535"
						bind:value={portInput}
						class={cn(portError && 'border-destructive focus-visible:ring-destructive')}
						oninput={() => {
							if (portError) portError = '';
						}}
						disabled={updateDomainMutation.isPending}
					/>
					{#if portError}
						<p class="text-sm text-destructive">{portError}</p>
					{/if}
				</div>
			{/if}

			<div class="flex items-center justify-end">
				<Button type="submit" disabled={updateDomainMutation.isPending || !isDirty}>
					{updateDomainMutation.isPending ? 'Updating...' : 'Save'}
				</Button>
			</div>
		</form>
	</CardContent>
</Card>
