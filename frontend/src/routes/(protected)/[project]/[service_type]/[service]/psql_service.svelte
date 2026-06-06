<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Card, CardContent } from '@/components/ui/card';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Skeleton } from '@/components/ui/skeleton';
	import { Eye, EyeOff } from '@lucide/svelte';
	import {
		useRedeployPsqlServiceMutation,
		useUpdatePsqlServiceMutation
	} from '@/features/services/mutation.svelte';
	import { useGetPsqlServiceDetailsQuery } from '@/features/services/query.svelte';
	import { createForm } from '@tanstack/svelte-form';
	import { z } from 'zod';
	import FormError from '@/components/services/FormError.svelte';
	import type { UpdatePsqlServicePayload } from '@/features/services/type';

	const { serviceID }: { serviceID: string } = $props();

	const serviceQuery = useGetPsqlServiceDetailsQuery(() => serviceID);
	const updatePsqlService = useUpdatePsqlServiceMutation(() => serviceID);
	const redeployPsqlService = useRedeployPsqlServiceMutation();

	let isPasswordVisible = $state(false);

	const form = createForm(() => ({
		defaultValues: {
			service_id: serviceID,
			db_name: '',
			db_user: '',
			db_password: ''
		} as UpdatePsqlServicePayload,
		onSubmit: ({ value }) => {
			updatePsqlService.mutate({
				service_id: serviceID,
				db_name: value.db_name.trim(),
				db_user: value.db_user.trim(),
				db_password: value.db_password
			});
		}
	}));

	$effect(() => {
		if (serviceQuery.data) {
			const { db_name, db_password, db_user } = serviceQuery.data;
			form.setFieldValue('db_name', db_name);
			form.setFieldValue('db_user', db_user);
			form.setFieldValue('db_password', db_password);
		}
	});
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
		<!-- {@const _ = hydrateForm(details)} -->

		<Card class="mt-4">
			<CardContent class="space-y-6 py-6">
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-lg font-semibold">{details.name}</h2>
						<p class="text-sm text-muted-foreground">{details.internal_url}</p>
					</div>
					<Button
						variant="outline"
						onclick={() => redeployPsqlService.mutate({ service_id: serviceID })}
						disabled={redeployPsqlService.isPending}
					>
						{redeployPsqlService.isPending ? 'Redeploying...' : 'Redeploy'}
					</Button>
				</div>

				<form
					class="flex flex-col gap-4"
					onsubmit={(e) => {
						e.preventDefault();
						e.stopPropagation();
						form.handleSubmit();
					}}
				>
					<form.Field
						name="db_name"
						validators={{ onChange: z.string().min(1, 'Database name is required') }}
					>
						{#snippet children(field)}
							<div class="space-y-1.5">
								<Label class="my-1" for={field.name}>Database Name</Label>
								<Input
									id={field.name}
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
									disabled={updatePsqlService.isPending}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="db_user"
						validators={{ onChange: z.string().min(1, 'Database user is required') }}
					>
						{#snippet children(field)}
							<div class="space-y-1.5">
								<Label class="my-1" for={field.name}>Database User</Label>
								<Input
									id={field.name}
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
									disabled={updatePsqlService.isPending}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="db_password"
						validators={{ onChange: z.string().min(8, 'Password must be at least 8 characters') }}
					>
						{#snippet children(field)}
							<div class="space-y-1.5">
								<Label class="my-1" for={field.name}>Database Password</Label>
								<div class="relative">
									<Input
										id={field.name}
										type={isPasswordVisible ? 'text' : 'password'}
										value={field.state.value}
										onblur={field.handleBlur}
										oninput={(e) => field.handleChange(e.currentTarget.value)}
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
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

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
