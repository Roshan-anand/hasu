<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import { useGetAllInstanceQuery } from '@/features/base/query.svelte';
	import { getBaseState } from '@/features/global/store.svelte';
	import * as Select from '@/components/ui/select';
	import { Blocks } from '@lucide/svelte';

	const base = getBaseState();

	const exp = /^(?!#\/(?:members|git|storage)?$).+$/;

	const { project, service, type } = $derived.by(() => {
		if (!exp.test(page.url.hash))
			return {
				project: null,
				service: null,
				type: null
			};

		const paths = page.url.hash.split('/');
		paths.shift();
		return {
			project: paths[0],
			type: paths[1],
			service: paths[2]
		};
	});

	const getAllInstance = useGetAllInstanceQuery(() => project);

	$effect(() => {
		if (getAllInstance.isSuccess) {
			getAllInstance.data.forEach((i) => {
				if (i.is_production) {
					base.setCurrentInstance(i.id, i.name);
				}
			});
		}
	});
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
			{#if base.currentInstance.id !== ''}
				<Breadcrumb.Item>
					<Select.Root type="single" value={base.currentInstance.name}>
						<Select.Trigger
							class="w-full h-fit border-none dark:bg-transparent bg-transparent focus:bg-transparent"
							id="git-branch-select"
						>
							{#if base.currentInstance.name}
								<span>{base.currentInstance.name}</span>
							{:else}
								<span>Select Instance</span>
							{/if}
						</Select.Trigger>
						<Select.Content>
							{#if getAllInstance.data}
								{#each getAllInstance.data as { id, name } (id)}
									<Select.Item
										value={name}
										label={name}
										onclick={() => base.setCurrentInstance(id, name)}>{name}</Select.Item
									>
								{/each}
							{/if}
						</Select.Content>
					</Select.Root>
				</Breadcrumb.Item>
			{:else}
				<Breadcrumb.Item>loading...</Breadcrumb.Item>
			{/if}
			{#if service && type !== 'new'}
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Link>{service}</Breadcrumb.Link>
				</Breadcrumb.Item>
			{/if}
		</Breadcrumb.List>
	</Breadcrumb.Root>
{/if}
