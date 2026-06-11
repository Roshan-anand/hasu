<script lang="ts">
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import FormError from '@/components/services/FormError.svelte';
	import {
		useProfileQuery,
		useUpdateProfileMutation,
		useChangePasswordMutation,
		AVATAR_OPTIONS
	} from '@/features/profile';

	const profileQuery = useProfileQuery();
	const updateProfile = useUpdateProfileMutation();
	const changePassword = useChangePasswordMutation();

	let selectedAvatar = $state('');
	let name = $state('');
	let email = $state('');
	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');

	// Derived validation errors (only show after the field is touched)
	const nameError = $derived.by(() => {
		if (!name) return undefined;
		const result = z.string().min(3, 'Name must be at least 3 characters').safeParse(name);
		return result.success ? undefined : result.error.issues[0].message;
	});

	const emailError = $derived.by(() => {
		if (!email) return undefined;
		const result = z.string().email('Please enter a valid email').safeParse(email);
		return result.success ? undefined : result.error.issues[0].message;
	});

	const oldPasswordError = $derived.by(() => {
		if (!oldPassword) return undefined;
		const result = z
			.string()
			.min(8, 'Password must be at least 8 characters')
			.safeParse(oldPassword);
		return result.success ? undefined : result.error.issues[0].message;
	});

	const newPasswordError = $derived.by(() => {
		if (!newPassword) return undefined;
		const result = z
			.string()
			.min(8, 'Password must be at least 8 characters')
			.safeParse(newPassword);
		return result.success ? undefined : result.error.issues[0].message;
	});

	const confirmPasswordError = $derived.by(() => {
		if (!confirmPassword) return undefined;
		return confirmPassword !== newPassword ? 'Passwords do not match' : undefined;
	});

	const passwordFormError = $derived.by(() => {
		if (!newPassword || !confirmPassword) return undefined;
		return newPassword !== confirmPassword ? 'Passwords do not match' : undefined;
	});

	const profileCanSubmit = $derived(
		name.trim().length >= 3 && z.string().email().safeParse(email).success
	);

	const passwordCanSubmit = $derived(
		z.string().min(8).safeParse(oldPassword).success &&
			z.string().min(8).safeParse(newPassword).success &&
			newPassword === confirmPassword
	);

	// populate form with profile data once loaded
	$effect(() => {
		if (profileQuery.isPending) return;
		const profile = profileQuery.data;
		if (!profile) return;
		console.log('Loaded profile trigger');
		name = profile.name;
		email = profile.email;
		selectedAvatar = profile.avatar || AVATAR_OPTIONS[0];
	});

	$effect(() => {
		if (profileQuery.isError) {
			toast.error('Failed to load profile');
		}
	});

	function handleProfileSubmit(e: Event) {
		e.preventDefault();
		if (!profileCanSubmit || updateProfile.isPending) return;
		updateProfile.mutate({
			name: name.trim(),
			email: email.trim(),
			avatar: selectedAvatar
		});
	}

	function handlePasswordSubmit(e: Event) {
		e.preventDefault();
		if (!passwordCanSubmit || changePassword.isPending) return;
		changePassword.mutate({
			old_password: oldPassword,
			new_password: newPassword
		});
	}
</script>

{#if profileQuery.isPending}
	<p class="text-muted-foreground">Loading profile...</p>
{:else if profileQuery.isError}
	<p class="text-destructive">Failed to load profile. Please try again later.</p>
{:else}
	<div class="mx-auto w-full">
		<!-- Profile Section: Avatar left  Name/Email right on md -->
		<section class="space-y-4">
			<h2 class="text-lg font-medium">Profile</h2>
			<form onsubmit={handleProfileSubmit}>
				<div class="flex flex-col md:flex-row md:gap-8">
					<!-- TODO : change avatar with actual avatar png -->
					<!-- Avatar – left column on md, top row on mobile -->
					<div class="flex flex-row flex-wrap gap-3 md:flex-col">
						{#each AVATAR_OPTIONS as avatarPath, i (i)}
							<Button
								type="button"
								variant="none"
								onclick={() => (selectedAvatar = avatarPath)}
								class={`size-15 rounded-full overflow-hidden border-2 p-1 cursor-pointer transition-all hover:scale-105
								${selectedAvatar === avatarPath ? 'border-primary ring-2 ring-primary/30' : 'border-border hover:border-muted-foreground'} `}
							>
								<img
									src={avatarPath}
									alt={avatarPath}
									class="size-full object-cover object-center"
								/>
							</Button>
						{/each}
					</div>

					<!-- Form fields – right column on md -->
					<div class="flex-1 space-y-4">
						<div class="grid gap-2">
							<Label for="profile-name">Display Name</Label>
							<Input id="profile-name" type="text" bind:value={name} />
							<FormError errors={nameError ? [nameError] : []} />
						</div>

						<div class="grid gap-2">
							<Label for="profile-email">Email</Label>
							<Input id="profile-email" type="email" bind:value={email} />
							<FormError errors={emailError ? [emailError] : []} />
						</div>

						<Button type="submit" disabled={!profileCanSubmit || updateProfile.isPending}>
							{updateProfile.isPending ? 'Saving...' : 'Save Profile'}
						</Button>
					</div>
				</div>
			</form>
		</section>

		<hr class="my-8" />

		<!-- Password Change – always stacked -->
		<section class="space-y-4">
			<h2 class="text-lg font-medium">Change Password</h2>
			<form onsubmit={handlePasswordSubmit} class="space-y-4">
				<div class="grid gap-2">
					<Label for="password-old">Current Password</Label>
					<Input id="password-old" type="password" bind:value={oldPassword} />
					<FormError errors={oldPasswordError ? [oldPasswordError] : []} />
				</div>

				<div class="grid gap-2">
					<Label for="password-new">New Password</Label>
					<Input id="password-new" type="password" bind:value={newPassword} />
					<FormError errors={newPasswordError ? [newPasswordError] : []} />
				</div>

				<div class="grid gap-2">
					<Label for="password-confirm">Confirm New Password</Label>
					<Input id="password-confirm" type="password" bind:value={confirmPassword} />
					<FormError errors={confirmPasswordError ? [confirmPasswordError] : []} />
				</div>

				{#if passwordFormError}
					<p class="text-sm text-destructive">{passwordFormError}</p>
				{/if}

				<Button
					type="submit"
					variant="secondary"
					disabled={!passwordCanSubmit || changePassword.isPending}
				>
					{changePassword.isPending ? 'Changing...' : 'Change Password'}
				</Button>
			</form>
		</section>
	</div>
{/if}
