<script lang="ts">
	import { Button } from '@/components/ui/button';
	import * as Dialog from '@/components/ui/dialog';
	import GitProviderField from '@/components/services/git-provider-field.svelte';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import * as Select from '@/components/ui/select';
	import { gitProviders, serviceTypes } from '@/features/services/const';
	import { normalizePathValue, validateAppGitForm } from '@/features/services/form';
	import {
		useCreateServiceMutation,
		useGetReposMutation
	} from '@/features/services/mutation.svelte';
	import { useGithubAppsQuery } from '@/features/services/query.svelte';
	import { getServiceState } from '@/features/services/store.svelte';
	import type { GithubApp, GitProviderKey, GitProviderOption } from '@/features/services/type';
	import { queryClient } from '@/query';
	import { createForm, revalidateLogic } from '@tanstack/svelte-form';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';
	import type { ServiceType } from '@/types';
	import { getUserState } from '@/features/global/store.svelte';

	const { currentOrg } = getUserState();
	const featureState = getServiceState();

	const githubAppsQuery = useGithubAppsQuery();
	const getReposMutation = useGetReposMutation();

	featureState.setAfterCreateSuccess(async ({ id, type }) => {
		await queryClient.invalidateQueries({ queryKey: ['services'] });
		featureState.closeCreateDialog();
		resetGitRepoSelection();
		form.reset();

		toast.success('Service created successfully');
		await goto(
			resolve('/(core)/[service_type]/[service_id]?tab=deployment', {
				service_type: type,
				service_id: id
			})
		);
	});

	const createServiceMutation = useCreateServiceMutation();

	const form = createForm(() => ({
		defaultValues: {
			name: '',
			type: 'app',
			git_provider: '' as GitProviderKey | '',
			git_app_id: '',
			git_repo_id: '',
			build_path: '/',
			watch_path: '/',
			db_name: '',
			db_user: '',
			db_password: '',
			image: ''
		},
		validationLogic: revalidateLogic(),
		validators: {
			onDynamic: ({ value }) => {
				const fields: Record<string, string> = {};

				switch (value.type) {
					case 'app':
						Object.assign(fields, validateAppGitForm(value) ?? {});
						break;

					case 'psql':
						if (value.db_name.trim() === '') fields.db_name = 'Database name is required';
						if (value.db_user.trim() === '') fields.db_user = 'Database user is required';
						if (value.db_password === '') fields.db_password = 'Database password is required';
						if (value.image.trim() === '') fields.image = 'Image is required';
						break;
				}
				return Object.keys(fields).length > 0 ? { fields } : undefined;
			}
		},
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

				const ghAppId = Number.parseInt(value.git_app_id, 10);
				if (Number.isNaN(ghAppId)) {
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
						git_provider: (value.git_provider || 'github') as GitProviderKey,
						gh_app_id: ghAppId,
						git_repo_id: value.git_repo_id,
						git_repo_name: selectedGithubRepo.full_name,
						git_repo_url: selectedGithubRepo.repo_url,
						default_branch: selectedGithubRepo.default_branch,
						build_path: buildPath,
						watch_path: watchPath
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

	const closeDialog = () => {
		if (createServiceMutation.isPending) return;
		featureState.closeCreateDialog();
	};

	const resetGitRepoSelection = () => {
		form.resetField('git_provider');
		form.resetField('git_app_id');
		form.resetField('git_repo_id');
		form.resetField('build_path');
		form.resetField('watch_path');
		featureState.githubApps = [];
		featureState.githubRepos = [];
	};

	const onServiceTypeChange = (type: ServiceType) => {
		const currentType = form.getFieldValue('type');
		form.setFieldValue('type', type);

		if (currentType === type) return;

		resetGitRepoSelection();

		if (type === 'app') {
			void githubAppsQuery.refetch();
			form.resetField('db_name');
			form.resetField('db_user');
			form.resetField('db_password');
			form.resetField('image');
		} else {
			form.resetField('git_provider');
			form.resetField('git_app_id');
			form.resetField('git_repo_id');
			form.resetField('build_path');
			form.resetField('watch_path');
		}
	};

	$effect(() => {
		if (!featureState.createDialogOpen) return;
		if (form.getFieldValue('type') !== 'app') return;
		if (currentOrg.id === '') return;
		if (githubAppsQuery.isFetching) return;
		if (featureState.githubApps.length > 0) return;

		void githubAppsQuery.refetch();
	});

	const selectGithubApp = (app: GithubApp) => {
		if (currentOrg.id === '' || createServiceMutation.isPending || getReposMutation.isPending)
			return;

		const githubProvider = gitProviders.find((provider) => provider.key === 'github');
		if (!githubProvider) return;

		form.setFieldValue('git_provider', 'github');
		form.setFieldValue('git_app_id', app.app_id.toString());
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
		form.setFieldValue('git_app_id', '');
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

	const getGithubAppName = (appId: string): string => {
		return featureState.githubApps.find((app) => app.app_id.toString() === appId)?.name ?? '';
	};

	const getGithubRepoName = (repoId: string): string => {
		return featureState.githubRepos.find((repo) => repo.id.toString() === repoId)?.full_name ?? '';
	};
</script>

<Dialog.Root bind:open={featureState.createDialogOpen}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/40" />
		<Dialog.Content
			class="fixed z-50 top-1/2 left-1/2 w-[92vw] max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-background p-5 shadow-lg"
		>
			<Dialog.Title class="text-lg font-semibold">Create Project</Dialog.Title>
			<Dialog.Description class="text-sm text-muted-foreground"
				>Create a project.</Dialog.Description
			>

			<form
				class="mt-4 space-y-4"
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
							{#if field.state.meta.errors.length}
								<p class="text-sm font-medium text-destructive">{field.state.meta.errors[0]}</p>
							{/if}
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
								onValueChange={(value) => onServiceTypeChange(value as ServiceType)}
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

				<form.Subscribe selector={(state) => state.values.type}>
					{#snippet children(currentType)}
						{#if currentType === 'app'}
							<form.Subscribe
								selector={(state) => ({
									gitProvider: state.values.git_provider,
									gitAppId: state.values.git_app_id,
									gitRepoId: state.values.git_repo_id
								})}
							>
								{#snippet children(gitState)}
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
												{#if field.state.meta.errors.length}
													<p class="text-sm font-medium text-destructive">
														{field.state.meta.errors[0]}
													</p>
												{/if}
											{/snippet}
										</form.Field>

										<form.Field name="git_app_id">
											{#snippet children(field)}
												<div class="space-y-1.5">
													<Label for="github-app-select">GitHub App</Label>
													{#if featureState.githubApps.length === 0}
														<div
															class="rounded-md border border-dashed p-3 text-sm text-muted-foreground"
														>
															No app connected.
															<a
																href={resolve('/(core)/git')}
																class="ml-1 underline underline-offset-4"
															>
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
																{getGithubAppName(gitState.gitAppId) || 'Select GitHub app'}
															</Select.Trigger>
															<Select.Content>
																{#each featureState.githubApps as app (app.app_id)}
																	<Select.Item value={app.app_id.toString()} label={app.name} />
																{/each}
															</Select.Content>
														</Select.Root>
													{/if}
													{#if field.state.meta.errors.length}
														<p class="text-sm font-medium text-destructive">
															{field.state.meta.errors[0]}
														</p>
													{/if}
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
														disabled={gitState.gitAppId === ''}
													>
														<Select.Trigger class="w-full" id="git-repo-select">
															{getGithubRepoName(gitState.gitRepoId) || 'Select repository'}
														</Select.Trigger>
														<Select.Content>
															{#each featureState.githubRepos as repo (repo.id)}
																<Select.Item value={repo.id.toString()} label={repo.full_name} />
															{/each}
														</Select.Content>
													</Select.Root>
													{#if field.state.meta.errors.length}
														<p class="text-sm font-medium text-destructive">
															{field.state.meta.errors[0]}
														</p>
													{/if}
												</div>
											{/snippet}
										</form.Field>

								<form.Field
									name="build_path"
											validators={{
												onChange: z.string().min(1, 'Build path is required'),
												onDynamic: ({ value, fieldApi }) => {
													if (fieldApi.form.getFieldValue('type') !== 'app') return undefined;
													return value.trim() === '' ? 'Build path is required' : undefined;
												}
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
													{#if field.state.meta.errors.length}
														<p class="text-sm font-medium text-destructive">
															{field.state.meta.errors[0]}
														</p>
													{/if}
												</div>
											{/snippet}
										</form.Field>

										<form.Field
											name="watch_path"
											validators={{
												onChange: z.string().min(1, 'Watch path is required'),
												onDynamic: ({ value, fieldApi }) => {
													if (fieldApi.form.getFieldValue('type') !== 'app') return undefined;
													return value.trim() === '' ? 'Watch path is required' : undefined;
												}
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
													{#if field.state.meta.errors.length}
														<p class="text-sm font-medium text-destructive">
															{field.state.meta.errors[0]}
														</p>
													{/if}
												</div>
											{/snippet}
										</form.Field>
									</div>
								{/snippet}
							</form.Subscribe>
						{/if}

						{#if currentType === 'psql'}
							<form.Field
								name="db_name"
								validators={{
									onChange: z.string().min(1, 'Database name is required'),
									onDynamic: ({ value, fieldApi }) => {
										if (fieldApi.form.getFieldValue('type') !== 'psql') return undefined;
										return value.trim() === '' ? 'Database name is required' : undefined;
									}
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
										{#if field.state.meta.errors.length}
											<p class="text-sm font-medium text-destructive">
												{field.state.meta.errors[0]}
											</p>
										{/if}
									</div>
								{/snippet}
							</form.Field>

							<form.Field
								name="db_user"
								validators={{
									onChange: z.string().min(1, 'Database user is required'),
									onDynamic: ({ value, fieldApi }) => {
										if (fieldApi.form.getFieldValue('type') !== 'psql') return undefined;
										return value.trim() === '' ? 'Database user is required' : undefined;
									}
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
										{#if field.state.meta.errors.length}
											<p class="text-sm font-medium text-destructive">
												{field.state.meta.errors[0]}
											</p>
										{/if}
									</div>
								{/snippet}
							</form.Field>

							<form.Field
								name="db_password"
								validators={{
									onChange: z.string().min(1, 'Database password is required'),
									onDynamic: ({ value, fieldApi }) => {
										if (fieldApi.form.getFieldValue('type') !== 'psql') return undefined;
										return value === '' ? 'Database password is required' : undefined;
									}
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
										{#if field.state.meta.errors.length}
											<p class="text-sm font-medium text-destructive">
												{field.state.meta.errors[0]}
											</p>
										{/if}
									</div>
								{/snippet}
							</form.Field>

							<form.Field
								name="image"
								validators={{
									onChange: z.string().min(1, 'Image is required'),
									onDynamic: ({ value, fieldApi }) => {
										if (fieldApi.form.getFieldValue('type') !== 'psql') return undefined;
										return value.trim() === '' ? 'Image is required' : undefined;
									}
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
										{#if field.state.meta.errors.length}
											<p class="text-sm font-medium text-destructive">
												{field.state.meta.errors[0]}
											</p>
										{/if}
									</div>
								{/snippet}
							</form.Field>
						{/if}
					{/snippet}
				</form.Subscribe>

				<form.Subscribe
					selector={(state) => ({ canSubmit: state.canSubmit, isSubmitting: state.isSubmitting })}
				>
					{#snippet children(state)}
						<div class="flex justify-end gap-2 pt-1">
							<Button
								variant="outline"
								type="button"
								onclick={closeDialog}
								disabled={createServiceMutation.isPending}
							>
								Cancel
							</Button>
							<Button
								type="submit"
								disabled={!state.canSubmit || state.isSubmitting || createServiceMutation.isPending}
							>
								{state.isSubmitting || createServiceMutation.isPending ? 'Creating...' : 'Create'}
							</Button>
						</div>
					{/snippet}
				</form.Subscribe>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
