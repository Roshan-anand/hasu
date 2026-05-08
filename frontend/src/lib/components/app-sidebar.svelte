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
	import { Blocks, Users, GitBranch } from '@lucide/svelte';
	import Organisation from './Organisation.svelte';
</script>

<Sidebar>
	<SidebarHeader>
		<SidebarMenu>
			<SidebarMenuItem>
				<SidebarMenuButton size="lg">
					<Organisation />
				</SidebarMenuButton>
			</SidebarMenuItem>
		</SidebarMenu>
	</SidebarHeader>
	<SidebarContent>
		<SidebarGroup>
			<SidebarGroupContent>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton onclick={() => goto(resolve('/(protected)/(core)'))}>
							<Blocks />
							<span>Projects</span>
						</SidebarMenuButton>
					</SidebarMenuItem>
					<SidebarMenuItem>
						<SidebarMenuButton onclick={() => goto(resolve('/(protected)/(core)/members'))}>
							<Users />
							<span>Members</span>
						</SidebarMenuButton>
					</SidebarMenuItem>
					<SidebarMenuItem>
						<SidebarMenuButton onclick={() => goto(resolve('/(protected)/(core)/git'))}>
							<GitBranch />
							<span>Git</span>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarGroupContent>
		</SidebarGroup>
	</SidebarContent>
	<SidebarFooter />
	<SidebarRail />
</Sidebar>
