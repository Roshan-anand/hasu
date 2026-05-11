<script lang="ts">
	import { createForm } from '@tanstack/svelte-form';
	import { z } from 'zod';
	import { toast } from 'svelte-sonner';
	import AuthBranding from '@/components/auth-branding.svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Checkbox } from '@/components/ui/checkbox';
	import { Label } from '@/components/ui/label';
	import { resolve } from '$app/paths';
	import { useRegisterMutation } from '@/features/auth/mutation.svelte';
	import FormError from '@/components/services/FormError.svelte';

	const register = useRegisterMutation();

	const form = createForm(() => ({
		defaultValues: {
			name: '',
			email: '',
			password: '',
			organization: '',
			rememberMe: false
		},
		onSubmit: async ({ value }) => {
			register.mutate({
				name: value.name,
				email: value.email,
				password: value.password,
				org_name: value.organization
			});
		}
	}));

	$effect(() => {
		if (register.isError) {
			toast.error(
				register.error?.message ?? 'An error occurred while registering. Please try again.'
			);
		}
	});
</script>

<div class="grid min-h-svh lg:grid-cols-2">
	<AuthBranding />
	<div class="flex items-center justify-center p-8">
		<div class="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-100">
			<div class="flex flex-col space-y-2 text-center">
				<h1 class="text-2xl font-semibold tracking-tight">Create an account</h1>
				<p class="text-sm text-muted-foreground">Enter your details below to create your account</p>
			</div>

			<form
				onsubmit={(e) => {
					e.preventDefault();
					e.stopPropagation();
					form.handleSubmit();
				}}
			>
				<div class="grid gap-4">
					<form.Field
						name="name"
						validators={{
							onChange: z.string().min(3, 'Name must be at least 3 characters')
						}}
					>
						{#snippet children(field)}
							<div class="grid gap-2">
								<Label class="my-1" for={field.name}>Name</Label>
								<Input
									id={field.name}
									name={field.name}
									type="text"
									placeholder="John Doe"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="email"
						validators={{
							onChange: z.email('Please enter a valid email')
						}}
					>
						{#snippet children(field)}
							<div class="grid gap-2">
								<Label class="my-1" for={field.name}>Email</Label>
								<Input
									id={field.name}
									name={field.name}
									type="email"
									placeholder="name@example.com"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
								/>
								{#if field.state.meta.errors.length}
									<p class="text-sm font-medium text-destructive">
										{field.state.meta.errors[0] ?? 'Invalid email'}
									</p>
								{/if}
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="password"
						validators={{
							onChange: z.string().min(8, 'Password must be at least 8 characters')
						}}
					>
						{#snippet children(field)}
							<div class="grid gap-2">
								<Label class="my-1" for={field.name}>Password</Label>
								<Input
									id={field.name}
									name={field.name}
									type="password"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="organization"
						validators={{
							onChange: z.string().min(3, 'organization must be at least 3 characters')
						}}
					>
						{#snippet children(field)}
							<div class="grid gap-2">
								<Label class="my-1" for={field.name}>organization</Label>
								<Input
									id={field.name}
									name={field.name}
									type="text"
									placeholder="Acme Inc."
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field name="rememberMe">
						{#snippet children(field)}
							<div class="flex items-center space-x-2 mt-2">
								<Checkbox
									id={field.name}
									checked={field.state.value}
									onchange={() => field.handleChange(!field.state.value)}
								/>
								<Label
									for={field.name}
									class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
								>
									Remember me
								</Label>
							</div>
						{/snippet}
					</form.Field>

					<form.Subscribe
						selector={(state) => ({
							canSubmit: state.canSubmit,
							isSubmitting: state.isSubmitting
						})}
					>
						{#snippet children(state)}
							<Button
								type="submit"
								class="w-full mt-2"
								disabled={!state.canSubmit || register.isPending}
							>
								{state.isSubmitting || register.isPending
									? 'Creating account...'
									: 'Create account'}
							</Button>
						{/snippet}
					</form.Subscribe>
				</div>
			</form>

			<p class="px-8 text-center text-sm text-muted-foreground">
				Already have an account?
				<Button variant="link" href={resolve('/login')}>Log in</Button>
			</p>
		</div>
	</div>
</div>
