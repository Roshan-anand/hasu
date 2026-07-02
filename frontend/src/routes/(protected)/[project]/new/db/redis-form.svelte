<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import * as Select from '@/components/ui/select';
	import { useCreateRedisServiceMutation } from '@/features/services';
	import { useGetOrphanVolumesByTypeQuery } from '@/features/base';
	import { createForm } from '@tanstack/svelte-form';
	import { z } from 'zod';
	import FormError from '@/components/services/FormError.svelte';
	import type { CreateRedisServiceBody } from '@/features/services';
	import { Eye, EyeOff } from '@lucide/svelte';

	const { data } = $props();
	const { projectName } = $derived(data);

	const createRedisServiceMutation = useCreateRedisServiceMutation(() => projectName);
	const orphanVolumesQuery = useGetOrphanVolumesByTypeQuery('redis');

	let isPasswordVisible = $state(false);
	const orphanVolumes = $derived(orphanVolumesQuery.data ?? []);

	const redisImages = [
		{ label: 'Redis 7 (alpine)', value: 'redis:7-alpine' },
		{ label: 'Redis 7', value: 'redis:7' },
		{ label: 'Redis 6 (alpine)', value: 'redis:6-alpine' },
		{ label: 'Redis 6', value: 'redis:6' }
	];

	const form = createForm(() => ({
		defaultValues: {
			name: '',
			password: 'app_password',
			image: 'redis:7-alpine',
			volume: ''
		} as CreateRedisServiceBody,
		onSubmit: ({ value }) => {
			createRedisServiceMutation.mutate(value);
		}
	}));
</script>

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
		validators={{
			onChange: z
				.string()
				.min(3, 'Service name must be at least 3 characters')
				.regex(/^\S*$/, 'Service name should not contain spaces. Use "-" instead.')
		}}
	>
		{#snippet children(field)}
			<div class="space-y-1.5">
				<Label class="my-1" for={field.name}>Service Name</Label>
				<Input
					id={field.name}
					placeholder="session-cache"
					value={field.state.value}
					onblur={field.handleBlur}
					oninput={(e) => field.handleChange(e.currentTarget.value)}
					disabled={createRedisServiceMutation.isPending}
				/>
				<FormError errors={field.state.meta.errors} />
			</div>
		{/snippet}
	</form.Field>

	<form.Field name="password">
		{#snippet children(field)}
			<div class="space-y-1.5">
				<Label class="my-1" for={field.name}>Password</Label>
				<div class="relative">
					<Input
						id={field.name}
						type={isPasswordVisible ? 'text' : 'password'}
						placeholder="••••••••"
						value={field.state.value}
						onblur={field.handleBlur}
						oninput={(e) => field.handleChange(e.currentTarget.value)}
						disabled={createRedisServiceMutation.isPending}
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
						disabled={createRedisServiceMutation.isPending}
					>
						{#if isPasswordVisible}
							<EyeOff class="h-4 w-4" />
						{:else}
							<Eye class="h-4 w-4" />
						{/if}
						<span class="sr-only">Toggle password visibility</span>
					</Button>
				</div>
				<p class="text-xs text-muted-foreground">Leave empty for no password.</p>
				<FormError errors={field.state.meta.errors} />
			</div>
		{/snippet}
	</form.Field>

	<form.Field name="image" validators={{ onChange: z.string().min(1, 'Image is required') }}>
		{#snippet children(field)}
			<div class="space-y-1.5">
				<Label class="my-1" for={field.name}>Redis Image</Label>
				<Select.Root
					type="single"
					value={field.state.value}
					onValueChange={(value) => field.handleChange(value)}
					disabled={createRedisServiceMutation.isPending}
				>
					<Select.Trigger class="w-full" id={field.name}>
						{redisImages.find((image) => image.value === field.state.value)?.label ||
							'Select image'}
					</Select.Trigger>
					<Select.Content>
						{#each redisImages as image (image.value)}
							<Select.Item value={image.value} label={image.label} />
						{/each}
					</Select.Content>
				</Select.Root>
				<FormError errors={field.state.meta.errors} />
			</div>
		{/snippet}
	</form.Field>

	<!-- Data source: single select, "New database" by default or pick an orphan volume to reattach -->
	<form.Field name="volume">
		{#snippet children(field)}
			<div class="space-y-1.5 rounded-lg border p-4">
				<Label class="my-1" for={field.name}>Data Source</Label>
				<Select.Root
					type="single"
					value={field.state.value}
					onValueChange={(value) => field.handleChange(value)}
					disabled={createRedisServiceMutation.isPending}
				>
					<Select.Trigger class="w-full" id={field.name}>
						{#if field.state.value === ''}
							New database
						{:else}
							{field.state.value}
						{/if}
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="" label="New database" />
						{#each orphanVolumes as vol (vol.id)}
							<Select.Item value={vol.volume} label={vol.volume} />
						{/each}
					</Select.Content>
				</Select.Root>
				{#if field.state.value !== ''}
					<p class="text-xs text-muted-foreground">
						Heads up: redis image versions must be compatible with the data on the volume. A
						mismatch may prevent the database from starting.
					</p>
				{/if}
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
					disabled={!state.canSubmit || state.isSubmitting || createRedisServiceMutation.isPending}
				>
					{state.isSubmitting || createRedisServiceMutation.isPending
						? 'Creating...'
						: 'Create Redis Service'}
				</Button>
			</div>
		{/snippet}
	</form.Subscribe>
</form>
