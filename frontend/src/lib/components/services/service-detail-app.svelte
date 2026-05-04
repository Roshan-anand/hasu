<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import * as Select from '@/components/ui/select';
	import GitProviderField from '@/components/services/git-provider-field.svelte';
	import { gitProviders } from '@/features/services/const';
	import { normalizePathValue, validateAppGitForm } from '@/features/services/form';
	import {
		useGetReposMutation,
		useUpdateAppServiceMutation
	} from '@/features/services/mutation.svelte';
	import { useGithubAppsQuery } from '@/features/services/query.svelte';
	import { getServiceState } from '@/features/services/store.svelte';
	import type { GithubApp, GitProviderKey, GitProviderOption } from '@/features/services/type';
	import { getUserState } from '@/features/global/store.svelte';
	import { queryClient } from '@/query';
	import type { AppService, ServiceBase } from '@/types';
	import { createForm, revalidateLogic } from '@tanstack/svelte-form';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';

	let { service } = $props<{ service: ServiceBase & AppService }>();
	const { currentOrg } = getUserState();
	const featureState = getServiceState();
	const githubAppsQuery = useGithubAppsQuery();
	const getReposMutation = useGetReposMutation();
	const updateServiceMutation = useUpdateAppServiceMutation();

	let lastFetchedAppId: number | null = null;

	const initialBuildPath = $derived(normalizePathValue(service.build_path));
	const initialWatchPath = $derived(normalizePathValue(service.watch_path));

	const form = createForm(() => ({
		defaultValues: {
			git_provider: service.git_provider as GitProviderKey,
			git_app_id: service.gh_app_id.toString(),
			git_repo_id: service.git_repo_id,
			default_branch: service.default_branch,
			build_path: initialBuildPath,
			watch_path: initialWatchPath
		},
		validationLogic: revalidateLogic(),
		validators: {
			onDynamic: ({ value }) => {
				const fields = validateAppGitForm(value);
				return fields ? { fields } : undefined;
			}
		},
		onSubmit: ({ value }) => {
			const selectedRepo = featureState.githubRepos.find(
				(repo) => repo.id.toString() === value.git_repo_id
			);

			if (!selectedRepo && value.git_repo_id !== service.git_repo_id) {
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

			updateServiceMutation.mutate(
				{
					service_id: service.id,
					git_provider: (value.git_provider || 'github') as GitProviderKey,
					gh_app_id: ghAppId,
					git_repo_id: value.git_repo_id,
					git_repo_name: selectedRepo?.full_name ?? service.git_repo_name,
					git_repo_url: selectedRepo?.repo_url ?? service.git_repo_url,
					default_branch: selectedRepo?.default_branch ?? service.default_branch,
					build_path: buildPath,
					watch_path: watchPath
				},
				{
					onSuccess: async (response) => {
						await queryClient.invalidateQueries({
							queryKey: ['service-details', 'app', service.id]
						});
						await queryClient.invalidateQueries({ queryKey: ['services'] });
						toast.success(response?.message || 'Service updated successfully');
					}
				}
			);
		}
	}));

	const resetGitRepoSelection = () => {
		form.resetField('git_app_id');
		form.resetField('git_repo_id');
		featureState.githubRepos = [];
	};

	const selectGithubApp = (app: GithubApp) => {
		if (currentOrg.id === '' || updateServiceMutation.isPending || getReposMutation.isPending)
			return;

		const githubProvider = gitProviders.find((provider) => provider.key === 'github');
		if (!githubProvider) return;

		form.setFieldValue('git_provider', 'github');
		form.setFieldValue('git_app_id', app.app_id.toString());
		form.setFieldValue('git_repo_id', '');
		featureState.githubRepos = [];
		lastFetchedAppId = app.app_id;

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
		if (provider.api === '' || currentOrg.id === '' || updateServiceMutation.isPending) return;

		form.setFieldValue('git_provider', provider.key);
		resetGitRepoSelection();

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
		const app = featureState.githubApps.find((item) => item.app_id.toString() === appId);
		if (app) return app.name;
		if (appId === service.gh_app_id.toString()) return `App ${service.gh_app_id}`;
		return '';
	};

	const getGithubRepoName = (repoId: string): string => {
		if (repoId === service.git_repo_id) return service.git_repo_name;
		return featureState.githubRepos.find((repo) => repo.id.toString() === repoId)?.full_name ?? '';
	};

	const hasChanges = (value: {
		git_provider: GitProviderKey | '';
		git_app_id: string;
		git_repo_id: string;
		default_branch: string;
		build_path: string;
		watch_path: string;
	}) => {
		return (
			value.git_provider !== service.git_provider ||
			value.git_app_id !== service.gh_app_id.toString() ||
			value.git_repo_id !== service.git_repo_id ||
			value.default_branch !== service.default_branch ||
			normalizePathValue(value.build_path) !== initialBuildPath ||
			normalizePathValue(value.watch_path) !== initialWatchPath
		);
	};

	$effect(() => {
		if (currentOrg.id === '') return;
		if (githubAppsQuery.isFetching) return;
		if (featureState.githubApps.length > 0) return;
		void githubAppsQuery.refetch();
	});

	$effect(() => {
		const appId = Number.parseInt(form.getFieldValue('git_app_id'), 10);
		if (Number.isNaN(appId)) return;
		if (getReposMutation.isPending) return;
		if (lastFetchedAppId === appId) return;

		const githubProvider = gitProviders.find((provider) => provider.key === 'github');
		if (!githubProvider) return;

		lastFetchedAppId = appId;
		getReposMutation.mutate({
			provider: githubProvider,
			appId
		});
	});
</script>

<div class="rounded-lg border bg-card text-card-foreground shadow-sm p-5 space-y-4">
	<div>
		<h1 class="text-xl font-semibold">{service.name}</h1>
		<p class="text-sm uppercase text-muted-foreground">{service.type}</p>
	</div>

	<p class="text-sm text-muted-foreground">{service.description || 'No description'}</p>

	<div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
		<div>
			<p class="text-muted-foreground">Service ID</p>
			<p class="font-medium break-all">{service.id}</p>
		</div>
		<div>
			<p class="text-muted-foreground">App Name</p>
			<p class="font-medium">{service.app_name}</p>
		</div>
	</div>

	<form
		class="space-y-3"
		onsubmit={(e) => {
			e.preventDefault();
			e.stopPropagation();
			form.handleSubmit();
		}}
	>
		<form.Field name="git_provider">
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label>Git Provider</Label>
					<GitProviderField
						value={field.state.value}
						disabled={currentOrg.id === '' ||
							getReposMutation.isPending ||
							updateServiceMutation.isPending}
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
				</div>
			{/snippet}
		</form.Field>

		<form.Field name="git_app_id">
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label for="github-app-select">GitHub App</Label>
					<Select.Root
						type="single"
						value={field.state.value}
						onValueChange={(value) => {
							field.handleChange(value);
							onGithubAppSelect(value);
						}}
						disabled={updateServiceMutation.isPending}
					>
						<Select.Trigger class="w-full" id="github-app-select">
							{getGithubAppName(field.state.value) || 'Select GitHub app'}
						</Select.Trigger>
						<Select.Content>
							{#each featureState.githubApps as app (app.app_id)}
								<Select.Item value={app.app_id.toString()} label={app.name} />
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
						disabled={form.getFieldValue('git_app_id') === ''}
					>
						<Select.Trigger class="w-full" id="git-repo-select">
							{getGithubRepoName(field.state.value) || 'Select repository'}
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
						disabled={updateServiceMutation.isPending}
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
						disabled={updateServiceMutation.isPending}
					/>
					{#if field.state.meta.errors.length}
						<p class="text-sm font-medium text-destructive">
							{field.state.meta.errors[0]}
						</p>
					{/if}
				</div>
			{/snippet}
		</form.Field>

		<form.Subscribe
			selector={(state) => ({
				values: state.values,
				canSubmit: state.canSubmit,
				isSubmitting: state.isSubmitting
			})}
		>
			{#snippet children(state)}
				<div class="flex justify-end gap-2 pt-2">
					<Button
						type="submit"
						disabled={!state.canSubmit ||
							state.isSubmitting ||
							updateServiceMutation.isPending ||
							!hasChanges(state.values)}
					>
						{state.isSubmitting || updateServiceMutation.isPending ? 'Saving...' : 'Save'}
					</Button>
				</div>
			{/snippet}
		</form.Subscribe>
	</form>
</div>
