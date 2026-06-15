<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Card, CardContent } from '@/components/ui/card';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { Eye, EyeOff, Copy, Play, Square } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import {
		useRedeployPsqlServiceMutation,
		useUpdatePsqlServiceMutation,
		useStopPredefServiceMutation,
		useStartPredefServiceMutation
	} from '@/features/services';
	import { useGetPsqlServiceDetailsQuery } from '@/features/services';
	import FormError from '@/components/services/FormError.svelte';

	const { serviceID }: { serviceID: string } = $props();

	const serviceQuery = useGetPsqlServiceDetailsQuery(() => serviceID);
	const updatePsqlService = useUpdatePsqlServiceMutation(() => serviceID);
	const redeployPsqlService = useRedeployPsqlServiceMutation();
	const stopService = useStopPredefServiceMutation(() => serviceID);
	const startService = useStartPredefServiceMutation(() => serviceID);

	const isRunning = $derived(serviceQuery.data?.status === 'running');
	const isPaused = $derived(serviceQuery.data?.status === 'paused');

	let isPasswordVisible = $state(false);

	let dbName = $state('');
	let dbUser = $state('');
	let dbPassword = $state('');

	let errors = $state<{ db_name?: string; db_user?: string; db_password?: string }>({});

	$effect(() => {
		if (serviceQuery.data) {
			const { db_name, db_password, db_user } = serviceQuery.data;
			dbName = db_name;
			dbUser = db_user;
			dbPassword = db_password;
		}
	});

	function validate(): boolean {
		const next: typeof errors = {};
		if (!dbName.trim()) next.db_name = 'Database name is required';
		if (!dbUser.trim()) next.db_user = 'Database user is required';
		if (!dbPassword || dbPassword.length < 8)
			next.db_password = 'Password must be at least 8 characters';
		errors = next;
		return Object.keys(next).length === 0;
	}

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (!validate()) return;
		updatePsqlService.mutate({
			service_id: serviceID,
			db_name: dbName.trim(),
			db_user: dbUser.trim(),
			db_password: dbPassword
		});
	}
</script>

<section class="p-4 max-w-3xl">
	<h1 class="text-xl font-semibold">PSQL Service</h1>

	{#if serviceQuery.isPending}
		<div class="mt-4 flex flex-col gap-4">
			<Skeleton class="h-40 w-full" />
			<Skeleton class="h-24 w-full" />
		</div>
	{:else if serviceQuery.isError}
		<p class="mt-4 text-red-500">Failed to load service details</p>
	{:else if serviceQuery.data}
		{@const details = serviceQuery.data}

		<Card class="mt-4">
			<CardContent class="space-y-6 py-6">
				<div class="flex items-center justify-between">
					<div class="space-y-1">
						<div class="flex items-center gap-3">
							<h2 class="text-lg font-semibold">{details.name}</h2>
							{#if isRunning}
								<span
									class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-600 dark:text-emerald-400"
								>
									<span class="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
									Running
								</span>
							{:else if isPaused}
								<span
									class="inline-flex items-center gap-1.5 rounded-full bg-amber-500/10 px-2.5 py-0.5 text-xs font-medium text-amber-600 dark:text-amber-400"
								>
									<span class="h-1.5 w-1.5 rounded-full bg-amber-500"></span>
									Paused
								</span>
							{/if}
						</div>
						<div class="flex items-center gap-2">
							<p class="text-sm text-muted-foreground">{details.internal_url}</p>
							<button
								type="button"
								onclick={async () => {
									try {
										await navigator.clipboard.writeText(details.internal_url);
										toast.success('Internal URL copied to clipboard');
									} catch {
										toast.error('Failed to copy to clipboard');
									}
								}}
								class="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
								title="Copy internal URL"
							>
								<Copy class="h-3.5 w-3.5" />
							</button>
						</div>
					</div>
					<div class="flex items-center gap-2">
						{#if isRunning}
							<Button
								variant="secondary"
								onclick={() => stopService.mutate()}
								disabled={stopService.isPending}
							>
								{#if stopService.isPending}
									Stopping...
								{:else}
									<Square class="mr-1.5 h-4 w-4" />
									Stop
								{/if}
							</Button>
						{:else if isPaused}
							<Button
								variant="default"
								onclick={() => startService.mutate()}
								disabled={startService.isPending}
							>
								{#if startService.isPending}
									Starting...
								{:else}
									<Play class="mr-1.5 h-4 w-4" />
									Start
								{/if}
							</Button>
						{/if}
						<Button
							variant="outline"
							onclick={() => redeployPsqlService.mutate({ service_id: serviceID })}
							disabled={redeployPsqlService.isPending || isPaused}
						>
							{redeployPsqlService.isPending ? 'Redeploying...' : 'Redeploy'}
						</Button>
					</div>
				</div>

				<form class="flex flex-col gap-4" onsubmit={handleSubmit}>
					<div class="space-y-1.5">
						<Label class="my-1" for="db_name">Database Name</Label>
						<Input id="db_name" bind:value={dbName} disabled={updatePsqlService.isPending} />
						<FormError errors={errors.db_name ? [errors.db_name] : []} />
					</div>

					<div class="space-y-1.5">
						<Label class="my-1" for="db_user">Database User</Label>
						<Input id="db_user" bind:value={dbUser} disabled={updatePsqlService.isPending} />
						<FormError errors={errors.db_user ? [errors.db_user] : []} />
					</div>

					<div class="space-y-1.5">
						<Label class="my-1" for="db_password">Database Password</Label>
						<div class="relative">
							<Input
								id="db_password"
								type={isPasswordVisible ? 'text' : 'password'}
								bind:value={dbPassword}
								disabled={updatePsqlService.isPending}
								class="pr-10"
							/>
							<Button
								variant="ghost"
								size="sm"
								class="absolute right-2 top-1/2 h-7 w-7 -translate-y-1/2 p-0"
								type="button"
								onclick={() => {
									isPasswordVisible = !isPasswordVisible;
								}}
								disabled={updatePsqlService.isPending}
							>
								{#if isPasswordVisible}
									<EyeOff class="h-4 w-4" />
								{:else}
									<Eye class="h-4 w-4" />
								{/if}
								<span class="sr-only">Toggle password visibility</span>
							</Button>
						</div>
						<FormError errors={errors.db_password ? [errors.db_password] : []} />
					</div>

					<div class="flex justify-end">
						<Button type="submit" disabled={updatePsqlService.isPending}>
							{updatePsqlService.isPending ? 'Saving...' : 'Save'}
						</Button>
					</div>
				</form>
			</CardContent>
		</Card>
	{:else}
		<p class="mt-4 text-muted-foreground">Service not found</p>
	{/if}
</section>
