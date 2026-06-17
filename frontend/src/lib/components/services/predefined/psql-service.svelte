<script lang="ts">
	import { Copy, Eye, EyeOff, X } from '@lucide/svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { toast } from 'svelte-sonner';
	import { useUpdatePsqlServiceMutation } from '@/features/services';
	import { useGetPsqlServiceDetailsQuery } from '@/features/services';
	import StreamLogs from '../stream-logs.svelte';
	import FormError from '@/components/services/FormError.svelte';
	import PsqlSettings from './psql-settings.svelte';

	let { serviceID, drawerOpen }: { serviceID: string; drawerOpen: boolean } = $props();

	const serviceQuery = useGetPsqlServiceDetailsQuery(() => serviceID);
	const updatePsqlService = useUpdatePsqlServiceMutation(() => serviceID);

	const isRunning = $derived(serviceQuery.data?.status === 'running');
	const isPaused = $derived(serviceQuery.data?.status === 'paused');

	let isPasswordVisible = $state(false);

	let dbName = $state('');
	let dbUser = $state('');
	let dbPassword = $state('');

	let originalDbName = $state('');
	let originalDbUser = $state('');
	let originalDbPassword = $state('');

	let errors = $state<{ db_name?: string; db_user?: string; db_password?: string }>({});

	$effect(() => {
		if (serviceQuery.data) {
			const { db_name, db_password, db_user } = serviceQuery.data;
			dbName = db_name;
			dbUser = db_user;
			dbPassword = db_password;
			originalDbName = db_name;
			originalDbUser = db_user;
			originalDbPassword = db_password;
		}
	});

	let hasChanges = $derived(
		dbName !== originalDbName || dbUser !== originalDbUser || dbPassword !== originalDbPassword
	);

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
		updatePsqlService.mutate(
			{
				service_id: serviceID,
				db_name: dbName.trim(),
				db_user: dbUser.trim(),
				db_password: dbPassword
			},
			{
				onSuccess: () => {
					originalDbName = dbName;
					originalDbUser = dbUser;
					originalDbPassword = dbPassword;
				}
			}
		);
	}
</script>

<section class="flex h-full max-w-full flex-col bg-background text-foreground">
	<header class="shrink-0 border-b border-border px-5 py-4">
		<div class="flex items-start justify-between gap-4">
			<div class="min-w-0 space-y-1">
				<p class="text-xs font-medium text-muted-foreground">Predefined Database Service</p>
				<h1 class="truncate text-lg font-semibold tracking-tight">PSQL Service</h1>
			</div>
			<Button
				variant="ghost"
				size="icon"
				class="-mr-1 h-8 w-8 shrink-0"
				onclick={() => (drawerOpen = false)}
				aria-label="Close service drawer"
			>
				<X class="h-4 w-4" />
			</Button>
		</div>
	</header>

	{#if serviceQuery.isPending}
		<div class="flex flex-1 flex-col gap-6 px-5 py-5">
			<Skeleton class="h-20 w-full" />
			<div class="space-y-3">
				<Skeleton class="h-8 w-full" />
				<Skeleton class="h-8 w-full" />
				<Skeleton class="h-8 w-full" />
			</div>
			<Skeleton class="h-56 w-full" />
		</div>
	{:else if serviceQuery.isError}
		<div class="px-5 py-5">
			<p
				class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
			>
				Failed to load service details
			</p>
		</div>
	{:else if serviceQuery.data}
		{@const { name, internal_url } = serviceQuery.data}

		<div class="flex min-h-0 flex-1 flex-col overflow-y-auto">
			<!-- Drawer layout keeps operator actions visible while using plain form rows instead of card wrappers. -->
			<div class="space-y-6 px-5 py-5">
				<section class="space-y-4">
					<div
						class="flex flex-col gap-4 min-[720px]:flex-row min-[720px]:items-start min-[720px]:justify-between"
					>
						<div class="min-w-0 space-y-3">
							<div class="flex flex-wrap items-center gap-2">
								<h2 class="truncate text-base font-semibold">{name}</h2>
								{#if isRunning}
									<span
										class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-300"
									>
										<span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
										Running
									</span>
								{:else if isPaused}
									<span
										class="inline-flex items-center gap-1.5 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-300"
									>
										<span class="h-1.5 w-1.5 rounded-full bg-amber-500"></span>
										Paused
									</span>
								{/if}
							</div>

							<div class="space-y-1.5">
								<p class="text-xs font-medium text-muted-foreground">Internal URL</p>
								<div class="flex min-w-0 items-center gap-2 rounded-md bg-muted px-2.5 py-2">
									<code class="min-w-0 flex-1 truncate text-xs text-foreground">{internal_url}</code
									>
									<button
										type="button"
										onclick={async () => {
											try {
												await navigator.clipboard.writeText(internal_url);
												toast.success('Internal URL copied to clipboard');
											} catch {
												toast.error('Failed to copy to clipboard');
											}
										}}
										class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
										title="Copy internal URL"
										aria-label="Copy internal URL"
									>
										<Copy class="h-3.5 w-3.5" />
									</button>
								</div>
							</div>
						</div>

						<PsqlSettings {serviceID} {name} />
					</div>
				</section>

				<form class="border-t border-border pt-5" onsubmit={handleSubmit}>
					<div class="mb-5 space-y-1">
						<h3 class="text-sm font-semibold">Connection settings</h3>
						<p class="text-sm text-muted-foreground">
							Changes are saved to service config. Redeploy when runtime values need to apply.
						</p>
					</div>

					<div class="space-y-4">
						<div class="grid gap-2 min-[720px]:grid-cols-[10rem_1fr] min-[720px]:items-start">
							<Label class="pt-1.5 text-sm" for="db_name">Database name</Label>
							<div class="space-y-1.5">
								<Input id="db_name" bind:value={dbName} disabled={updatePsqlService.isPending} />
								<FormError errors={errors.db_name ? [errors.db_name] : []} />
							</div>
						</div>

						<div class="grid gap-2 min-[720px]:grid-cols-[10rem_1fr] min-[720px]:items-start">
							<Label class="pt-1.5 text-sm" for="db_user">Database user</Label>
							<div class="space-y-1.5">
								<Input id="db_user" bind:value={dbUser} disabled={updatePsqlService.isPending} />
								<FormError errors={errors.db_user ? [errors.db_user] : []} />
							</div>
						</div>

						<div class="grid gap-2 min-[720px]:grid-cols-[10rem_1fr] min-[720px]:items-start">
							<Label class="pt-1.5 text-sm" for="db_password">Database password</Label>
							<div class="space-y-1.5">
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
										class="absolute right-1 top-1/2 h-7 w-7 -translate-y-1/2 p-0"
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
						</div>
					</div>

					<div class="mt-6 flex justify-end border-t border-border pt-4">
						<Button type="submit" disabled={updatePsqlService.isPending || !hasChanges}>
							{updatePsqlService.isPending ? 'Saving...' : 'Save changes'}
						</Button>
					</div>
				</form>
			</div>

			<StreamLogs url={`/api/service/logs?service_id=${serviceID}`} open={drawerOpen} />
		</div>
	{:else}
		<div class="px-5 py-5">
			<p class="text-sm text-muted-foreground">Service not found</p>
		</div>
	{/if}
</section>
