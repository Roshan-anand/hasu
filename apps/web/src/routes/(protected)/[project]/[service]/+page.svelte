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
	<section class="flex-1 px-4 py-2">
		<NavigationMenu.Root viewport={false} class="max-w-full block mb-6">
			<NavigationMenu.List class="border-b justify-start gap-4">
				{#each NavItems as item (item.label)}
					<NavigationMenu.Item class="hover:bg-transparent">
						<NavigationMenu.Link
							href={resolve(`/(protected)/[project]/[service]?tab=${item.tab}`, {
								project: projectName,
								service: serviceName
							})}
							data-active={tab == item.tab || (tab == undefined && item.tab == '')}
							class="text-muted-foreground cursor-pointer bg-transparent hover:bg-transparent underline-offset-12 hover:text-foreground data-active:underline  data-active:bg-transparent data-active:text-foreground px-0"
						>
							{item.label}
						</NavigationMenu.Link>
					</NavigationMenu.Item>
				{/each}
			</NavigationMenu.List>
		</NavigationMenu.Root>

		{#if serviceName === ''}
			<p class="text-muted-foreground">Missing service in URL</p>
		{:else if tab === 'deployment'}
			<AppDeployments {serviceID} />
		{:else if tab === 'env'}
			<p class="text-muted-foreground">Environment variables tab content goes here</p>
			<AppEnv {serviceID} />
		{:else if tab === 'settings'}
			<AppSettings {serviceID} {serviceName} />
		{:else}
			<AppHome {serviceID} project={projectName} />
		{/if}
	</section>
{/if}
