<script lang="ts">
	import { resolve } from '$app/paths';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import * as Select from '@/components/ui/select';
	import { Blocks } from '@lucide/svelte';
	import { getInstanceState } from '@/features/instance';
	import { page } from '$app/state';

	const instance = getInstanceState();
	const { project, service } = $derived(page.params);
</script>

{#if project}
	<Breadcrumb.Root>
		<Breadcrumb.List>
			<Breadcrumb.Item>
				<Breadcrumb.Link href={resolve('/')}>
					<Blocks size={16} />
				</Breadcrumb.Link>
			</Breadcrumb.Item>
			<Breadcrumb.Separator />
			<Breadcrumb.Item>
				<Breadcrumb.Link
					href={resolve('/(protected)/[project]', {
						project: project
					})}>{project}</Breadcrumb.Link
				>
			</Breadcrumb.Item>
			<Breadcrumb.Separator />
			{#if instance.current.id}
				<Breadcrumb.Item>
					<Select.Root type="single" value={instance.current.name}>
						<Select.Trigger
							class="w-full h-fit border-none dark:bg-transparent bg-transparent focus:bg-transparent"
							id="git-branch-select"
						>
							{#if instance.current.name}
								<span>{instance.current.name}</span>
							{:else}
								<span>Select Instance</span>
							{/if}
						</Select.Trigger>
						<Select.Content>
							{#if instance.all}
								{#each instance.all as { id, name } (id)}
									<Select.Item
										value={name}
										label={name}
										onclick={() => instance.setCurrent(id, name)}>{name}</Select.Item
									>
								{/each}
							{/if}
						</Select.Content>
					</Select.Root>
				</Breadcrumb.Item>
			{:else}
				<Breadcrumb.Item>loading...</Breadcrumb.Item>
			{/if}
			{#if service}
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Link>{service}</Breadcrumb.Link>
				</Breadcrumb.Item>
			{/if}
		</Breadcrumb.List>
	</Breadcrumb.Root>
{/if}
