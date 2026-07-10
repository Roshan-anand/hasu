<script lang="ts">
	import { Button } from '@/components/ui/button';
	import { Checkbox } from '@/components/ui/checkbox';
	import GitProviderField from '@/components/services/git-provider-field.svelte';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import * as Select from '@/components/ui/select';
	import { useCreateServiceMutation, useGetReposMutation } from '@/features/services';
	import type { CreateAppServiceForm, GitProviderKey } from '@/features/services';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod';
	import FormError from '@/components/services/FormError.svelte';
	import { useGithubAppsQuery } from '@/features/git';
	import * as Collapsible from '@/components/ui/collapsible';
	import { ChevronRight, ChevronDown } from '@lucide/svelte';
	import SecretTextarea from '@/components/services/secret-textarea.svelte';
	import { createForm } from '@tanstack/svelte-form';
	import Icon from '@iconify/svelte';
	import { GitProvidersList } from '@/features/services';

	const { data } = $props();
	const { projectName } = $derived(data);

	let environmentOpen = $state(false);
	let buildSettingOpen = $state(false);

	const githubAppsQuery = useGithubAppsQuery();
	const getReposMutation = useGetReposMutation();
	const createServiceMutation = useCreateServiceMutation(() => projectName);

	const form = createForm(() => ({
		defaultValues: {
			name: '',
			git_provider: 'github',
			gh_app_id: 0,
			gh_repo_id: 0,
			default_branch: '',
			build_path: '/',
			watch_path: '/',
			public: true,
			port: 80,
			env: '',
			build_secrets: '',
			docker_build: {
				file_path: '',
				context_path: '',
				build_stage: ''
			}
		} as CreateAppServiceForm,
		onSubmit: ({ value }) => {
			const selectedGithubRepo = getReposMutation.data?.find(
				(repo) => repo.id === value.gh_repo_id
			);

			if (!selectedGithubRepo) {
				toast.error('Please select a repository');
				return;
			}

			if (value.gh_app_id === 0) {
				toast.error('Please select a GitHub app');
				return;
			}

			createServiceMutation.mutate(value);
		}
	}));

	// trigger the getrepo for the slected github app
	const onGithubAppSelect = (appId: string) => {
		const app = githubAppsQuery.data?.find((item) => item.app_id.toString() === appId);
		if (!app) return;
		if (createServiceMutation.isPending || getReposMutation.isPending) return;

		const githubProvider = GitProvidersList.get('github');
		if (!githubProvider) return;

		form.setFieldValue('gh_repo_id', 0);
		form.setFieldValue('default_branch', '');

		getReposMutation.mutate({
			appId: app.app_id,
			provider: githubProvider
		});
	};

	// reset and refetch app list of the selected git provider
	const fetchGitProvider = (key: GitProviderKey) => {
		if (createServiceMutation.isPending) return;
		const provider = GitProvidersList.get(key);
		if (!provider) return;

		form.setFieldValue('gh_app_id', 0);
		form.setFieldValue('gh_repo_id', 0);
		form.setFieldValue('default_branch', '');

		if (key === 'github' && githubAppsQuery.data?.length === 0) void githubAppsQuery.refetch();
	};

	// Sync branch selection with repo choice, defaulting to repo's default branch.
	const onRepoSelect = (repoId: number) => {
		const repo = getReposMutation.data?.find((r) => r.id === repoId);
		if (!repo) return;

		form.setFieldValue('gh_repo_id', repoId);
		const repoName = getReposMutation.data?.find((repo) => repo.id == repoId)?.name;
		form.setFieldValue('name', repoName || '');
		const defaultBranch = getReposMutation.data?.find((repo) => repo.id == repoId)?.default_branch;
		form.setFieldValue('default_branch', defaultBranch || '');
	};

	const getGithubAppName = (appId: number) =>
		githubAppsQuery.data?.find((app) => app.app_id === appId)?.name;
</script>

<section class="mx-auto w-full max-w-3xl p-4 md:p-6">
	<h1>New Project</h1>

	<form
		class="mt-4 flex flex-col gap-6"
		onsubmit={(e) => {
			e.preventDefault();
			e.stopPropagation();
			form.handleSubmit();
		}}
	>
		<form.Field name="git_provider">
			{#snippet children(field)}
				<Label class="my-1">Import from a git provider</Label>
				<GitProviderField
					value={field.state.value}
					onSelect={(key) => {
						field.handleChange(key);
						fetchGitProvider(key);
					}}
				/>
			{/snippet}
		</form.Field>

		<form.Field name="gh_app_id">
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label class="my-1" for="github-app-select">GitHub App</Label>
					{#if githubAppsQuery.data && githubAppsQuery.data.length === 0}
						<div class="rounded-md border border-dashed p-3 text-sm text-muted-foreground">
							No app connected.
							<a href={resolve('/git')} class="ml-1 underline underline-offset-4"> Go to Git </a>
						</div>
					{:else}
						<Select.Root
							type="single"
							value={field.state.value.toString()}
							onValueChange={(value) => {
								field.handleChange(parseInt(value));
								onGithubAppSelect(value);
							}}
							disabled={createServiceMutation.isPending}
						>
							<Select.Trigger class="w-full" id="github-app-select">
								{getGithubAppName(field.state.value) || 'Select GitHub app'}
							</Select.Trigger>
							<Select.Content>
								{#each githubAppsQuery.data as app (app.app_id)}
									<Select.Item value={app.app_id.toString()} label={app.name} />
								{/each}
							</Select.Content>
						</Select.Root>
					{/if}
					<FormError errors={field.state.meta.errors} />
				</div>
			{/snippet}
		</form.Field>

		<form.Subscribe
			selector={(state) => ({
				ghAppID: state.values.gh_app_id,
				gitProvider: state.values.git_provider,
				ghRepoID: state.values.gh_repo_id
			})}
		>
			{#snippet children({ ghAppID, gitProvider, ghRepoID })}
				<form.Field name="gh_repo_id">
					{#snippet children(field)}
						<div class="space-y-1.5">
							<Label class="my-1" for="git-repo-select">Repository</Label>
							<Select.Root
								type="single"
								value={field.state.value.toString()}
								onValueChange={(value) => {
									const id = parseInt(value);
									field.handleChange(id);
									onRepoSelect(id);
								}}
								disabled={ghAppID === 0}
							>
								<Select.Trigger class="w-full h-fit" id="git-repo-select">
									{#if getReposMutation.data}
										<div class="flex items-center gap-3 p-2 py-4">
											<Icon
												icon={GitProvidersList.get(gitProvider)?.icon || 'icon-park-outline:dot'}
												width="20"
												height="20"
												class="size-4"
											/>
											{#if field.state.value == 0}
												<span>Select repository</span>
											{/if}
											{#each getReposMutation.data as repo (repo.id)}
												{#if repo.id === field.state.value}
													<span class="text-sm text-muted-foreground">
														{repo.full_name}
													</span>
													<span class="flex items-center gap-1 text-muted-foreground">
														<Icon
															icon="meteor-icons:git-branch"
															width="20"
															height="20"
															class="size-4"
														/>
														{repo.default_branch}
													</span>
												{/if}
											{/each}
										</div>
									{:else}
										<span class="text-sm text-muted-foreground">
											{getReposMutation.isPending ? 'Loading repos...' : 'Select repository'}
										</span>
									{/if}
								</Select.Trigger>
								<Select.Content>
									{#if getReposMutation.data}
										{#each getReposMutation.data as repo (repo.id)}
											<Select.Item value={repo.id.toString()} label={repo.full_name} />
										{/each}
									{/if}
								</Select.Content>
							</Select.Root>
							<FormError errors={field.state.meta.errors} />
						</div>
					{/snippet}
				</form.Field>

				<form.Field name="default_branch">
					{#snippet children(field)}
						<div class="space-y-1.5">
							<Label class="my-1" for="git-branch-select">Branch</Label>
							<Select.Root
								type="single"
								value={field.state.value}
								onValueChange={(value) => field.handleChange(value)}
								disabled={ghAppID === 0 || ghRepoID === 0}
							>
								<Select.Trigger class="w-full h-fit" id="git-branch-select">
									{#if getReposMutation.data}
										<div class="flex items-center gap-3 p-2 py-4">
											<Icon icon="meteor-icons:git-branch" width="20" height="20" class="size-4" />
											{#if field.state.value === ''}
												<span>Select branch</span>
											{:else}
												<span class="text-sm text-muted-foreground">{field.state.value}</span>
											{/if}
										</div>
									{:else}
										<span class="text-sm text-muted-foreground">
											{getReposMutation.isPending ? 'Loading branches...' : 'Select branch'}
										</span>
									{/if}
								</Select.Trigger>
								<Select.Content>
									{#if getReposMutation.data}
										{#each getReposMutation.data as repo (repo.id)}
											{#if repo.id === ghRepoID}
												{#each repo.branches as branch (branch)}
													<Select.Item value={branch} label={branch} />
												{/each}
											{/if}
										{/each}
									{/if}
								</Select.Content>
							</Select.Root>
							<FormError errors={field.state.meta.errors} />
						</div>
					{/snippet}
				</form.Field>
			{/snippet}
		</form.Subscribe>

		<form.Field
			name="name"
			validators={{
				onChange: z
					.string()
					.min(3, 'Service name must be at least 3 characters')
					.regex(/^[^/]*$/, "app name must not include '/'")
			}}
		>
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label class="my-1" for={field.name}>Service Name</Label>
					<Input
						id={field.name}
						placeholder="Payments API"
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
			name="build_path"
			validators={{
				onChange: z.string().min(1, 'Build path is required')
			}}
		>
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label class="my-1" for={field.name}>Build Path</Label>
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
					<Label class="my-1" for={field.name}>Watch Path</Label>
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

		<form.Field name="public">
			{#snippet children(field)}
				<div class="flex items-center space-x-2">
					<Checkbox
						id={field.name}
						checked={field.state.value}
						onchange={() => field.handleChange(!field.state.value)}
						disabled={createServiceMutation.isPending}
					/>
					<Label
						for={field.name}
						class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
					>
						Make it public
					</Label>
				</div>
			{/snippet}
		</form.Field>

		<form.Field
			name="port"
			validators={{
				onChange: z
					.number()
					.int('Port must be an integer')
					.min(1, 'Port must be between 1 and 65535')
					.max(65535, 'Port must be between 1 and 65535')
			}}
		>
			{#snippet children(field)}
				<div class="space-y-1.5">
					<Label class="my-1" for={field.name}>Container Port</Label>
					<Input
						id={field.name}
						type="number"
						placeholder="80"
						value={field.state.value}
						onblur={field.handleBlur}
						oninput={(e) => field.handleChange(Number(e.currentTarget.value))}
						disabled={createServiceMutation.isPending}
					/>
					<p class="text-xs text-muted-foreground">
						The port your application listens on inside the container.
					</p>
					<FormError errors={field.state.meta.errors} />
				</div>
			{/snippet}
		</form.Field>

		<Collapsible.Root bind:open={environmentOpen} class="rounded-md border shadow-sm">
			<Collapsible.Trigger class="flex w-full items-center gap-2 font-medium p-2">
				{#if environmentOpen}
					<ChevronDown class="h-4 w-4" />
				{:else}
					<ChevronRight class="h-4 w-4" />
				{/if}
				Environment
			</Collapsible.Trigger>
			<Collapsible.Content class="space-y-4 p-2">
				<form.Field name="env">
					{#snippet children(field)}
						<div class="space-y-1.5">
							<SecretTextarea
								title="Environment Variables (Build & Runtime)"
								name={field.name}
								value={field.state.value}
								onblur={field.handleBlur}
								oninput={(e) => field.handleChange(e.currentTarget.value)}
								submitPending={createServiceMutation.isPending}
							/>
							<p class="text-xs text-muted-foreground">
								Available during build and when container runs.
							</p>
						</div>
					{/snippet}
				</form.Field>

				<form.Field name="build_secrets">
					{#snippet children(field)}
						<SecretTextarea
							title="Build Secrets"
							name={field.name}
							value={field.state.value}
							onblur={field.handleBlur}
							oninput={(e) => field.handleChange(e.currentTarget.value)}
							submitPending={createServiceMutation.isPending}
						/>
					{/snippet}
				</form.Field>
			</Collapsible.Content>
		</Collapsible.Root>

		<Collapsible.Root bind:open={buildSettingOpen} class="rounded-md border shadow-sm">
			<Collapsible.Trigger class="flex w-full items-center gap-2 font-medium p-2">
				{#if buildSettingOpen}
					<ChevronDown class="h-4 w-4" />
				{:else}
					<ChevronRight class="h-4 w-4" />
				{/if}
				Build setting
			</Collapsible.Trigger>
			<Collapsible.Content class="space-y-4 p-2">
				<form.Field name="docker_build.file_path">
					{#snippet children(field)}
						<div class="space-y-1.5">
							<Label class="my-1" for={field.name}>Docker File Path</Label>
							<Input
								id={field.name}
								type="text"
								value={field.state.value}
								onblur={field.handleBlur}
								oninput={(e) => field.handleChange(e.currentTarget.value)}
								disabled={createServiceMutation.isPending}
							/>
						</div>
					{/snippet}
				</form.Field>

				<form.Field name="docker_build.context_path">
					{#snippet children(field)}
						<div class="space-y-1.5">
							<Label class="my-1" for={field.name}>Docker Context Path</Label>
							<Input
								id={field.name}
								type="text"
								value={field.state.value}
								onblur={field.handleBlur}
								oninput={(e) => field.handleChange(e.currentTarget.value)}
								disabled={createServiceMutation.isPending}
							/>
						</div>
					{/snippet}
				</form.Field>

				<form.Field name="docker_build.build_stage">
					{#snippet children(field)}
						<div class="space-y-1.5">
							<Label class="my-1" for={field.name}>Docker Build Stage</Label>
							<Input
								id={field.name}
								type="text"
								value={field.state.value}
								onblur={field.handleBlur}
								oninput={(e) => field.handleChange(e.currentTarget.value)}
								disabled={createServiceMutation.isPending}
							/>
						</div>
					{/snippet}
				</form.Field>
			</Collapsible.Content>
		</Collapsible.Root>

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
</section>
