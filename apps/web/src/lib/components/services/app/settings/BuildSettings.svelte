<script lang="ts">
	import { useUpdateBuildSettingsMutation } from '@/features/services';
	import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { normalizePathValue } from '@/utils';

	let {
		serviceID,
		initialBuildPath = $bindable(''),
		initialWatchPath = $bindable(''),
		initialDockerFilepath = $bindable('Dockerfile'),
		initialDockerContextpath = $bindable('.'),
		initialDockerBuildstage = $bindable('')
	}: {
		serviceID: string;
		initialBuildPath?: string;
		initialWatchPath?: string;
		initialDockerFilepath?: string;
		initialDockerContextpath?: string;
		initialDockerBuildstage?: string;
	} = $props();

	const updateMutation = useUpdateBuildSettingsMutation(() => serviceID);

	let buildPath = $state(initialBuildPath);
	let watchPath = $state(initialWatchPath);
	let dockerFilepath = $state(initialDockerFilepath);
	let dockerContextpath = $state(initialDockerContextpath);
	let dockerBuildstage = $state(initialDockerBuildstage);

	$effect(() => {
		buildPath = initialBuildPath;
		watchPath = initialWatchPath;
		dockerFilepath = initialDockerFilepath;
		dockerContextpath = initialDockerContextpath;
		dockerBuildstage = initialDockerBuildstage;
	});

	const isDirty = $derived(
		buildPath !== initialBuildPath ||
			watchPath !== initialWatchPath ||
			dockerFilepath !== initialDockerFilepath ||
			dockerContextpath !== initialDockerContextpath ||
			dockerBuildstage !== initialDockerBuildstage
	);

	function handleSave() {
		if (!isDirty || updateMutation.isPending) return;

		updateMutation.mutate({
			service_id: serviceID,
			build_path: normalizePathValue(buildPath),
			watch_path: normalizePathValue(watchPath),
			docker_filepath: dockerFilepath.trim(),
			docker_contextpath: dockerContextpath.trim(),
			docker_buildstage: dockerBuildstage.trim()
		});
	}
</script>

<Card>
	<CardHeader>
		<CardTitle class="text-lg">Build Settings</CardTitle>
	</CardHeader>
	<CardContent>
		<form
			class="flex flex-col gap-4"
			onsubmit={(e) => {
				e.preventDefault();
				handleSave();
			}}
		>
			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<Label for="build-path">Build Path</Label>
					<Input
						id="build-path"
						placeholder="/"
						bind:value={buildPath}
						disabled={updateMutation.isPending}
					/>
					<p class="text-xs text-muted-foreground">Monorepo subdirectory to build from.</p>
				</div>

				<div class="space-y-1.5">
					<Label for="watch-path">Watch Path</Label>
					<Input
						id="watch-path"
						placeholder="/"
						bind:value={watchPath}
						disabled={updateMutation.isPending}
					/>
					<p class="text-xs text-muted-foreground">Monorepo subdirectory to watch for changes.</p>
				</div>
			</div>

			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<Label for="docker-filepath">Dockerfile Path</Label>
					<Input
						id="docker-filepath"
						placeholder="Dockerfile"
						bind:value={dockerFilepath}
						disabled={updateMutation.isPending}
					/>
				</div>

				<div class="space-y-1.5">
					<Label for="docker-contextpath">Docker Context Path</Label>
					<Input
						id="docker-contextpath"
						placeholder="."
						bind:value={dockerContextpath}
						disabled={updateMutation.isPending}
					/>
				</div>
			</div>

			<div class="space-y-1.5">
				<Label for="docker-buildstage">Build Stage</Label>
				<Input
					id="docker-buildstage"
					placeholder="Leave empty for default"
					bind:value={dockerBuildstage}
					disabled={updateMutation.isPending}
				/>
				<p class="text-xs text-muted-foreground">
					Target build stage in a multi-stage Dockerfile. Leave empty for default.
				</p>
			</div>

			<div class="flex items-center justify-end pt-2">
				<Button type="submit" disabled={updateMutation.isPending || !isDirty}>
					{updateMutation.isPending ? 'Updating...' : 'Save'}
				</Button>
			</div>
		</form>
	</CardContent>
</Card>
