<script lang="ts">
	import { createForm } from '@tanstack/svelte-form';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';
	import AuthBranding from '@/components/auth-branding.svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Checkbox } from '@/components/ui/checkbox';
	import { Label } from '@/components/ui/label';
	import { resolve } from '$app/paths';
	import { useLoginMutation } from '@/features/auth/mutation.svelte';
	import FormError from '@/components/services/FormError.svelte';

	const login = useLoginMutation();

	const form = createForm(() => ({
		defaultValues: {
			email: '',
			password: '',
			rememberMe: false
		},
		onSubmit: async ({ value }) => {
			login.mutate({ email: value.email, password: value.password });
		}
	}));

	$effect(() => {
		if (login.isError)
			toast.error(login.error?.message ?? 'An error occurred while logging in. Please try again.');
	});
</script>

<div class="grid min-h-svh lg:grid-cols-2">
	<AuthBranding />
	<div class="flex items-center justify-center p-8">
		<div class="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-87.5">
			<div class="flex flex-col space-y-2 text-center">
				<h1 class="text-2xl font-semibold tracking-tight">Log in to your account</h1>
				<p class="text-sm text-muted-foreground">Enter your email below to log in</p>
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
						name="email"
						validators={{
							onChange: z.email('Please enter a valid email')
						}}
					>
						{#snippet children(field)}
							<div class="grid gap-2">
								<Label for={field.name}>Email</Label>
								<Input
									id={field.name}
									name={field.name}
									type="email"
									placeholder="name@example.com"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
								/>
								<FormError errors={field.state.meta.errors} />
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
								<div class="flex items-center">
									<Label for={field.name}>Password</Label>
								</div>
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

					<form.Field name="rememberMe">
						{#snippet children(field)}
							<div class="flex items-center space-x-2">
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
							<Button type="submit" class="w-full" disabled={!state.canSubmit || login.isPending}>
								{state.isSubmitting || login.isPending ? 'Logging in...' : 'Log in'}
							</Button>
						{/snippet}
					</form.Subscribe>
				</div>
			</form>

			<p class="px-8 text-center text-sm text-muted-foreground">
				Don't have an account?
				<Button variant="link" href={resolve('/register')}>Sign up</Button>
			</p>
		</div>
	</div>
</div>
