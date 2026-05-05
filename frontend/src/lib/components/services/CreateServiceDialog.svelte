<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import GitProviderField from '@/components/services/git-provider-field.svelte';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Textarea } from '@/components/ui/textarea';
	import * as Select from '@/components/ui/select';
	import { gitProviders, serviceTypes } from '@/features/services/const';
	import { normalizePathValue } from '@/features/services/form';
	import {
		useCreateServiceMutation,
		useGetReposMutation
	} from '@/features/services/mutation.svelte';
	import { getServiceState } from '@/features/services/store.svelte';
	import type { CreateServiceForm, GithubApp, GitProviderOption } from '@/features/services/type';
	import { createForm } from '@tanstack/svelte-form';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';
	import { getUserState } from '@/features/global/store.svelte';
	import FormError from './FormError.svelte';
	import { useGithubAppsQuery } from '@/features/git/query.svelte';

	const { currentOrg } = getUserState();
	const featureState = getServiceState();

	const githubAppsQuery = useGithubAppsQuery();
	const getReposMutation = useGetReposMutation();

	const createServiceMutation = useCreateServiceMutation();

	const form = createForm(() => ({
		defaultValues: {
			type: 'app',
			name: '',
			git_provider: 'github',
			gh_app_id: '',
			git_repo_id: '',
			git_repo_name: '',
			git_repo_url: '',
			build_path: '/',
			watch_path: '/',
			env: '',
			build_args: '',
			build_secrets: ''
		} as CreateServiceForm,
		onSubmit: ({ value }) => {
			if (currentOrg.id === '') {
				toast.error('Please select an organization');
				return;
			}

			if (value.type === 'app') {
				const selectedGithubRepo = featureState.githubRepos.find(
					(repo) => repo.id.toString() === value.git_repo_id
				);

				if (!selectedGithubRepo) {
					toast.error('Please select a repository');
					return;
				}

				if (value.gh_app_id === '') {
					toast.error('Please select a GitHub app');
					return;
				}

				const buildPath = normalizePathValue(value.build_path);
				const watchPath = normalizePathValue(value.watch_path);

				createServiceMutation.mutate({
					type: 'app',
					body: {
						org_id: currentOrg.id,
						name: value.name.trim(),
						git_provider: value.git_provider,
						gh_app_id: value.gh_app_id,
						git_repo_id: value.git_repo_id,
						git_repo_name: selectedGithubRepo.full_name,
						git_repo_url: selectedGithubRepo.repo_url,
						default_branch: selectedGithubRepo.default_branch,
						build_path: buildPath,
						watch_path: watchPath,
						env: value.env,
						build_args: value.build_args,
						build_secrets: value.build_secrets
					}
				});
				return;
			}

			createServiceMutation.mutate({
				type: 'psql',
				body: {
					org_id: currentOrg.id,
					name: value.name.trim(),
					db_name: value.db_name.trim(),
					db_user: value.db_user.trim(),
					db_password: value.db_password,
					image: value.image.trim()
				}
			});
		}
	}));

	const selectGithubApp = (app: GithubApp) => {
		if (currentOrg.id === '' || createServiceMutation.isPending || getReposMutation.isPending)
			return;

		const githubProvider = gitProviders.find((provider) => provider.key === 'github');
		if (!githubProvider) return;

		form.setFieldValue('git_provider', 'github');
		form.setFieldValue('gh_app_id', app.app_id.toString());
		form.setFieldValue('git_repo_id', '');
		featureState.githubRepos = [];

		getReposMutation.mutate({
			provider: githubProvider,
			appId: app.app_id
		});
	};

	const onGithubAppSelect = (appId: string) => {
		const app = featureState.githubApps.find((item) => item.app_id.toString() === appId);
		if (!app) return;

		selectGithubApp(app);
	};

	const fetchGitProvider = (provider: GitProviderOption) => {
		if (provider.api === '' || currentOrg.id === '' || createServiceMutation.isPending) return;

		form.setFieldValue('git_provider', provider.key);
		form.setFieldValue('gh_app_id', '');
		form.setFieldValue('git_repo_id', '');
		featureState.githubRepos = [];

		if (provider.key === 'github' && featureState.githubApps.length === 0) {
			void githubAppsQuery.refetch();
		}
	};

	const onRepoSelect = (repoId: string) => {
		const repo = featureState.githubRepos.find((r) => r.id.toString() === repoId);
		if (!repo) return;

		form.setFieldValue('git_repo_id', repoId);
	};

	const getGithubAppName = (appId: string): string =>
		featureState.githubApps.find((app) => app.app_id.toString() === appId)?.name ?? '';

	const getGithubRepoName = (repoId: string): string =>
		featureState.githubRepos.find((repo) => repo.id.toString() === repoId)?.full_name ?? '';
</script>

<Dialog.Root bind:open={featureState.createDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-background" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[90vw] max-w-150 -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Create Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground"
				>Create a project.</Dialog.Description
			>

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
							<Label for={field.name}>Service Name</Label>
							<Input
								id={field.name}
								placeholder="Payments Database"
								value={field.state.value}
								onblur={field.handleBlur}
								oninput={(e) => field.handleChange(e.currentTarget.value)}
								disabled={createServiceMutation.isPending}
							/>
							<FormError errors={field.state.meta.errors} />
						</div>
					{/snippet}
				</form.Field>

				<form.Field name="type">
					{#snippet children(field)}
						<div class="space-y-1.5">
							<Label for={field.name}>Service Type</Label>
							<Select.Root
								type="single"
								value={field.state.value}
								onValueChange={(value) => {
									field.handleChange(value as 'app' | 'psql');
									if (value === 'app' && currentOrg.id !== '' && !githubAppsQuery.isFetching) {
										void githubAppsQuery.refetch();
									}
								}}
							>
								<Select.Trigger class="w-full" id={field.name}>
									{serviceTypes.find((item) => item.value === field.state.value)?.label}
								</Select.Trigger>
								<Select.Content>
									{#each serviceTypes as item (item.value)}
										<Select.Item value={item.value} label={item.label} />
									{/each}
								</Select.Content>
							</Select.Root>
						</div>
					{/snippet}
				</form.Field>

				{#if form.getFieldValue('type') === 'app'}
					<div class="space-y-2">
						<form.Field name="git_provider">
							{#snippet children(field)}
								<Label>Git</Label>
								<GitProviderField
									value={field.state.value}
									disabled={currentOrg.id === '' ||
										getReposMutation.isPending ||
										createServiceMutation.isPending}
									onSelect={(provider) => {
										field.handleChange(provider.key);
										fetchGitProvider(provider);
									}}
								/>
							{/snippet}
						</form.Field>

						<form.Field name="gh_app_id">
							{#snippet children(field)}
								<div class="space-y-1.5">
									<Label for="github-app-select">GitHub App</Label>
									{#if featureState.githubApps.length === 0}
										<div class="rounded-md border border-dashed p-3 text-sm text-muted-foreground">
											No app connected.
											<a href={resolve('/(core)/git')} class="ml-1 underline underline-offset-4">
												Go to Git
											</a>
										</div>
									{:else}
										<Select.Root
											type="single"
											value={field.state.value}
											onValueChange={(value) => {
												field.handleChange(value);
												onGithubAppSelect(value);
											}}
											disabled={createServiceMutation.isPending}
										>
											<Select.Trigger class="w-full" id="github-app-select">
												{getGithubAppName(form.getFieldValue('gh_app_id')) || 'Select GitHub app'}
											</Select.Trigger>
											<Select.Content>
												{#each featureState.githubApps as app (app.app_id)}
													<Select.Item value={app.app_id.toString()} label={app.name} />
												{/each}
											</Select.Content>
										</Select.Root>
									{/if}
									<FormError errors={field.state.meta.errors} />
								</div>
							{/snippet}
						</form.Field>

						<form.Field name="git_repo_id">
							{#snippet children(field)}
								<div class="space-y-1.5">
									<Label for="git-repo-select">Repository</Label>
									<Select.Root
										type="single"
										value={field.state.value}
										onValueChange={(value) => {
											field.handleChange(value);
											onRepoSelect(value);
										}}
										disabled={form.getFieldValue('gh_app_id') === ''}
									>
										<Select.Trigger class="w-full" id="git-repo-select">
											{getGithubRepoName(form.getFieldValue('git_repo_id')) || 'Select repository'}
										</Select.Trigger>
										<Select.Content>
											{#each featureState.githubRepos as repo (repo.id)}
												<Select.Item value={repo.id.toString()} label={repo.full_name} />
											{/each}
										</Select.Content>
									</Select.Root>
									<FormError errors={field.state.meta.errors} />
								</div>
							{/snippet}
						</form.Field>

						<form.Field
							name="build_path"
							validators={{
								onChange: z.string().min(1, 'Build path is required')
							}}
						>
							{#snippet children(field)}
								<div class="space-y-1.5">
									<Label for={field.name}>Build Path</Label>
									<Input
										id={field.name}
										placeholder="/"
										value={field.state.value}
										onblur={field.handleBlur}
										oninput={(e) => field.handleChange(e.currentTarget.value)}
										disabled={createServiceMutation.isPending}
									/>
									<FormError errors={field.state.meta.errors} />
								</div>
							{/snippet}
						</form.Field>

						<form.Field
							name="watch_path"
							validators={{
								onChange: z.string().min(1, 'Watch path is required')
							}}
						>
							{#snippet children(field)}
								<div class="space-y-1.5">
									<Label for={field.name}>Watch Path</Label>
									<Input
										id={field.name}
										placeholder="/"
										value={field.state.value}
										onblur={field.handleBlur}
										oninput={(e) => field.handleChange(e.currentTarget.value)}
										disabled={createServiceMutation.isPending}
									/>
									<FormError errors={field.state.meta.errors} />
								</div>
							{/snippet}
						</form.Field>

						<form.Field name="env">
							{#snippet children(field)}
								<div class="space-y-1.5">
									<Label for={field.name}>Environment</Label>
									<Textarea
										id={field.name}
										spellcheck={false}
										placeholder="KEY=value"
										value={field.state.value}
										onblur={field.handleBlur}
										oninput={(e) => field.handleChange(e.currentTarget.value)}
										disabled={createServiceMutation.isPending}
									/>
									<FormError errors={field.state.meta.errors} />
								</div>
							{/snippet}
						</form.Field>

						<form.Field name="build_args">
							{#snippet children(field)}
								<div class="space-y-1.5">
									<Label for={field.name}>Build Args</Label>
									<Textarea
										id={field.name}
										spellcheck={false}
										placeholder="KEY=value"
										value={field.state.value}
										onblur={field.handleBlur}
										oninput={(e) => field.handleChange(e.currentTarget.value)}
										disabled={createServiceMutation.isPending}
									/>
									<FormError errors={field.state.meta.errors} />
								</div>
							{/snippet}
						</form.Field>

						<form.Field name="build_secrets">
							{#snippet children(field)}
								<div class="space-y-1.5">
									<Label for={field.name}>Build Secrets</Label>
									<Textarea
										id={field.name}
										spellcheck={false}
										placeholder="KEY=value"
										value={field.state.value}
										onblur={field.handleBlur}
										oninput={(e) => field.handleChange(e.currentTarget.value)}
										disabled={createServiceMutation.isPending}
									/>
									<FormError errors={field.state.meta.errors} />
								</div>
							{/snippet}
						</form.Field>
					</div>
				{/if}

				{#if form.getFieldValue('type') === 'psql'}
					<form.Field
						name="db_name"
						validators={{
							onChange: z.string().min(1, 'Database name is required')
						}}
					>
						{#snippet children(field)}
							<div class="space-y-1.5">
								<Label for={field.name}>Database Name</Label>
								<Input
									id={field.name}
									placeholder="payments"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
									disabled={createServiceMutation.isPending}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="db_user"
						validators={{
							onChange: z.string().min(1, 'Database user is required')
						}}
					>
						{#snippet children(field)}
							<div class="space-y-1.5">
								<Label for={field.name}>Database User</Label>
								<Input
									id={field.name}
									placeholder="postgres"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
									disabled={createServiceMutation.isPending}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="db_password"
						validators={{
							onChange: z.string().min(1, 'Database password is required')
						}}
					>
						{#snippet children(field)}
							<div class="space-y-1.5">
								<Label for={field.name}>Database Password</Label>
								<Input
									id={field.name}
									type="password"
									placeholder="********"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
									disabled={createServiceMutation.isPending}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>

					<form.Field
						name="image"
						validators={{
							onChange: z.string().min(1, 'Image is required')
						}}
					>
						{#snippet children(field)}
							<div class="space-y-1.5">
								<Label for={field.name}>Image</Label>
								<Input
									id={field.name}
									placeholder="postgres:16"
									value={field.state.value}
									onblur={field.handleBlur}
									oninput={(e) => field.handleChange(e.currentTarget.value)}
									disabled={createServiceMutation.isPending}
								/>
								<FormError errors={field.state.meta.errors} />
							</div>
						{/snippet}
					</form.Field>
				{/if}

				<form.Subscribe
					selector={(state) => ({ canSubmit: state.canSubmit, isSubmitting: state.isSubmitting })}
				>
					{#snippet children(state)}
						<div class="flex justify-end gap-2 pt-1">
							<Button
								class="w-full"
								type="submit"
								disabled={!state.canSubmit || state.isSubmitting || createServiceMutation.isPending}
							>
								{state.isSubmitting || createServiceMutation.isPending ? 'Deploying...' : 'Deploy'}
							</Button>
						</div>
					{/snippet}
				</form.Subscribe>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
