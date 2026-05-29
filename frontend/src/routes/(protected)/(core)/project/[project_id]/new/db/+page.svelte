<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import * as Select from '@/components/ui/select';
	import { useCreatePsqlServiceMutation } from '@/features/services/mutation.svelte';
	import { createForm } from '@tanstack/svelte-form';
	import { z } from 'zod';
	import FormError from '@/components/services/FormError.svelte';
	import type { CreatePsqlServiceBody } from '@/features/services/type';
	import { Eye, EyeOff } from '@lucide/svelte';

	const { data } = $props();
	const projectId = $derived(data.project_id);

	const createPsqlServiceMutation = useCreatePsqlServiceMutation();

	// AI: Toggle visibility for the password field.
	let isPasswordVisible = $state(false);

	// Predefined Postgres images to keep selection consistent.
	const psqlImages = [
		{ label: 'Postgres 16 (alpine)', value: 'postgres:16-alpine' },
		{ label: 'Postgres 16', value: 'postgres:16' },
		{ label: 'Postgres 15 (alpine)', value: 'postgres:15-alpine' },
		{ label: 'Postgres 15', value: 'postgres:15' },
		{ label: 'Postgres 14 (alpine)', value: 'postgres:14-alpine' },
		{ label: 'Postgres 14', value: 'postgres:14' }
	];

	const form = createForm(() => ({
		defaultValues: {
			name: '',
			db_name: 'app_db',
			db_user: 'app_user',
			db_password: 'app_password',
			image: 'postgres:16-alpine'
		} as CreatePsqlServiceBody,
		onSubmit: ({ value }) => {
			createPsqlServiceMutation.mutate({
				project_id: projectId,
				name: value.name.trim(),
				db_name: value.db_name.trim(),
				db_user: value.db_user.trim(),
				db_password: value.db_password,
				image: value.image.trim()
			});
		}
	}));
</script>

<section class="mx-auto w-full max-w-3xl p-4 md:p-6">
	<h1>New Database Service</h1>

	<form
		class="mt-4 flex flex-col gap-6"
		onsubmit={(e) => {
			e.preventDefault();
			e.stopPropagation();
			form.handleSubmit();
		}}
	>
		<form.Field
			name="name"
			validators={{ onChange: z.string().min(3, 'Service name must be at least 3 characters') }}
		>
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label class="my-1" for={field.name}>Service Name</Label>
					<Input
						id={field.name}
						placeholder="payments-db"
						value={field.state.value}
						onblur={field.handleBlur}
						oninput={(e) => field.handleChange(e.currentTarget.value)}
						disabled={createPsqlServiceMutation.isPending}
					/>
					<FormError errors={field.state.meta.errors} />
				</div>
			{/snippet}
		</form.Field>

		<form.Field
			name="db_name"
			validators={{ onChange: z.string().min(1, 'Database name is required') }}
		>
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label class="my-1" for={field.name}>Database Name</Label>
					<Input
						id={field.name}
						placeholder="app_db"
						value={field.state.value}
						onblur={field.handleBlur}
						oninput={(e) => field.handleChange(e.currentTarget.value)}
						disabled={createPsqlServiceMutation.isPending}
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
						placeholder="app_user"
						value={field.state.value}
						onblur={field.handleBlur}
						oninput={(e) => field.handleChange(e.currentTarget.value)}
						disabled={createPsqlServiceMutation.isPending}
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
							placeholder="••••••••"
							value={field.state.value}
							onblur={field.handleBlur}
							oninput={(e) => field.handleChange(e.currentTarget.value)}
							disabled={createPsqlServiceMutation.isPending}
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
							disabled={createPsqlServiceMutation.isPending}
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

		<form.Field name="image" validators={{ onChange: z.string().min(1, 'Image is required') }}>
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label class="my-1" for={field.name}>Postgres Image</Label>
					<Select.Root
						type="single"
						value={field.state.value}
						onValueChange={(value) => field.handleChange(value)}
						disabled={createPsqlServiceMutation.isPending}
					>
						<Select.Trigger class="w-full" id={field.name}>
							{psqlImages.find((image) => image.value === field.state.value)?.label ||
								'Select image'}
						</Select.Trigger>
						<Select.Content>
							{#each psqlImages as image (image.value)}
								<Select.Item value={image.value} label={image.label} />
							{/each}
						</Select.Content>
					</Select.Root>
					<FormError errors={field.state.meta.errors} />
				</div>
			{/snippet}
		</form.Field>

		<form.Subscribe
			selector={(state) => ({ canSubmit: state.canSubmit, isSubmitting: state.isSubmitting })}
		>
			{#snippet children(state)}
				<div class="flex justify-end gap-2 pt-1">
					<Button
						class="w-full"
						type="submit"
						disabled={!state.canSubmit || state.isSubmitting || createPsqlServiceMutation.isPending}
					>
						{state.isSubmitting || createPsqlServiceMutation.isPending
							? 'Creating...'
							: 'Create Database Service'}
					</Button>
				</div>
			{/snippet}
		</form.Subscribe>
	</form>
</section>
