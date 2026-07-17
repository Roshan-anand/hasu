<script lang="ts">
	import SecretTextarea from '@/components/services/secret-textarea.svelte';
	import { Button } from '@/components/ui/button';
	import { useUpdateEnvMutation } from '@/features/services';
	import { useGetServiceEnvQuery } from '@/features/services';
	import AppDependency from './app-dependency.svelte';

	let { serviceID }: { serviceID: string } = $props();

	let env = $state<string>('');
	let buildSecrets = $state<string>('');

	const getEnvQuery = useGetServiceEnvQuery(() => serviceID);
	const updateEnv = useUpdateEnvMutation(() => serviceID);

	const handleUpdateEnv = () => {
		updateEnv.mutate({
			service_id: serviceID,
			env,
			build_secrets: buildSecrets
		});
	};
</script>

<section class="flex flex-col gap-5">
	{#if getEnvQuery.data}
		<div class="justify-end flex">
			<Button onclick={handleUpdateEnv} disabled={updateEnv.isPending}>
				{updateEnv.isPending ? 'updating ...' : 'update'}
			</Button>
		</div>
		<div class="space-y-1.5">
			<SecretTextarea
				title="Environment Variables (Build & Runtime)"
				name="env"
				value={getEnvQuery.data.env.join('\n')}
				oninput={(e) => (env = e.currentTarget.value)}
				submitPending={updateEnv.isPending}
			/>
			<p class="text-xs text-muted-foreground">Available during build and when container runs.</p>
			<AppDependency {serviceID} />
		</div>
		<SecretTextarea
			title="Build Secrets"
			name="build-secrets"
			value={getEnvQuery.data.build_secrets.join('\n')}
			oninput={(e) => (buildSecrets = e.currentTarget.value)}
			submitPending={updateEnv.isPending}
		/>
	{:else}
		<p>no env found</p>
	{/if}
</section>
