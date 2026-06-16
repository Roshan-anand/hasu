<script lang="ts">
	import { AppDeployments, AppHome, AppSettings, AppEnv } from '@/components/services/app';
	import * as NavigationMenu from '@/components/ui/navigation-menu';
	import { NavItems } from '@/features/services';
	import { useGetServiceIDQuery } from '@/features/services';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	const { data } = $props();
	const { serviceName, projectName, tab } = $derived(data);

	const getServiceID = useGetServiceIDQuery(() => serviceName);

	$effect(() => {
		if (getServiceID.isError)
			goto(
				resolve('/(protected)/[project]', {
					project: projectName
				})
			);
	});
</script>

{#if getServiceID.isPending}
	<p>loading....</p>
{:else if getServiceID.data}
	{@const serviceID = getServiceID.data}
	<section class="p-2 flex-1">
		<div class="mb-2">
			<NavigationMenu.Root viewport={false} class="w-full max-w-full">
				<NavigationMenu.List class="flex-1 w-full rounded-lg bg-card p-1">
					{#each NavItems as item (item.label)}
						<NavigationMenu.Item>
							<NavigationMenu.Link
								href={resolve(`/(protected)/[project]/[service]?tab=${item.tab}`, {
									project: projectName,
									service: serviceName
								})}
								data-active={tab == item.tab || (tab == undefined && item.tab == '')}
								class="cursor-pointer px-3 py-2"
							>
								{item.label}
							</NavigationMenu.Link>
						</NavigationMenu.Item>
					{/each}
				</NavigationMenu.List>
			</NavigationMenu.Root>
		</div>

		{#if serviceName === ''}
			<p class="text-muted-foreground">Missing service in URL</p>
		{:else if tab === 'deployment'}
			<AppDeployments {serviceID} />
		{:else if tab === 'env'}
			<p class="text-muted-foreground">Environment variables tab content goes here</p>
			<AppEnv {serviceID} />
		{:else if tab === 'settings'}
			<AppSettings {serviceID} />
		{:else}
			<AppHome {serviceID} project={projectName} />
		{/if}
	</section>
{/if}
