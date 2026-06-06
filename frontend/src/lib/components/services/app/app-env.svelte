<script lang="ts">
	import SecretTextarea from '@/components/services/secret-textarea.svelte';
	import { Button } from '@/components/ui/button';
	import { useUpdateEnvMutation } from '@/features/services/mutation.svelte';
	import { useGetServiceEnvQuery } from '@/features/services/query.svelte';

	let { serviceID }: { serviceID: string } = $props();

	let env = $state<string>('');
	let buildArgs = $state<string>('');
	let buildSecrets = $state<string>('');

	const getEnvQuery = useGetServiceEnvQuery(() => serviceID);
	const updateEnv = useUpdateEnvMutation(() => serviceID);

	const handleUpdateEnv = () => {
		updateEnv.mutate({
			service_id: serviceID,
			env: env.split('\n'),
			build_args: buildArgs.split('\n'),
			build_secrets: buildSecrets.split('\n')
		});
	};
</script>

{#if getEnvQuery.data}
	<SecretTextarea
		title="Environment Variables"
		name="env"
		value={getEnvQuery.data.env.join('\n')}
		oninput={(e) => (env = e.currentTarget.value)}
		submitPending={updateEnv.isPending}
	/>
	<SecretTextarea
		title="Build Args"
		name="build-args"
		value={getEnvQuery.data.build_args.join('\n')}
		oninput={(e) => (buildArgs = e.currentTarget.value)}
		submitPending={updateEnv.isPending}
	/>
	<SecretTextarea
		title="Build Secrets"
		name="build-secrets"
		value={getEnvQuery.data.build_secrets.join('\n')}
		oninput={(e) => (buildSecrets = e.currentTarget.value)}
		submitPending={updateEnv.isPending}
	/>

	<Button onclick={handleUpdateEnv} disabled={updateEnv.isPending}>
		{updateEnv.isPending ? 'updating ...' : 'update'}
	</Button>
{:else}
	<p>no env found</p>
{/if}
