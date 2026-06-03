<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import {
		Sidebar,
		SidebarContent,
		SidebarFooter,
		SidebarGroup,
		SidebarGroupContent,
		SidebarHeader,
		SidebarMenu,
		SidebarMenuButton,
		SidebarMenuItem,
		SidebarRail
	} from '@/components/ui/sidebar';
	import { Blocks, Users, GitBranch, Database } from '@lucide/svelte';
	import { page } from '$app/state';
	import type { ResolvedPathname } from '$app/types';
	import Organization from './Organization.svelte';

	type AppSidebarItem = {
		hash: RegExp;
		route: ResolvedPathname;
		name: string;
		icon: typeof Blocks;
	};

	const sidebarItems: AppSidebarItem[] = [
		{
			hash: /^(?!#\/(?:members|git|storage)$).+/,
			route: resolve('/(protected)/(core)'),
			name: 'Projects',
			icon: Blocks
		},
		{
			hash: /^#\/members$/,
			route: resolve('/(protected)/(core)/members'),
			name: 'Members',
			icon: Users
		},
		{ hash: /^#\/git$/, route: resolve('/(protected)/(core)/git'), name: 'Git', icon: GitBranch },
		{
			hash: /^#\/storage$/,
			route: resolve('/(protected)/(core)/storage'),
			name: 'Storage',
			icon: Database
		}
	];
</script>

<Sidebar collapsible="icon">
	<SidebarHeader>
		<SidebarMenu>
			<SidebarMenuItem>
				<Organization />
			</SidebarMenuItem>
		</SidebarMenu>
	</SidebarHeader>
	<SidebarContent>
		<SidebarGroup>
			<SidebarGroupContent>
				<SidebarMenu class="gap-3">
					{#each sidebarItems as { name, hash, icon: Icon, route } (hash)}
						<SidebarMenuItem>
							<!-- eslint-disable svelte/no-navigation-without-resolve -->
							<SidebarMenuButton
								class={`${hash.test(page.url.hash) && 'bg-sidebar-accent text-sidebar-primary hover:text-sidebar-primary'}`}
								onclick={() => goto(route)}
							>
								<Icon />
								<span>{name}</span>
							</SidebarMenuButton>
						</SidebarMenuItem>
					{/each}
				</SidebarMenu>
			</SidebarGroupContent>
		</SidebarGroup>
	</SidebarContent>
	<SidebarFooter />
	<SidebarRail />
</Sidebar>
