<script lang="ts">
	import type { AppService, ServiceBase } from '@/types';

	// Display-only view for app service details.
	let { service }: { service: ServiceBase & AppService } = $props();
</script>

<div class="rounded-lg border bg-card text-card-foreground shadow-sm p-5 space-y-4">
	<div>
		<h1 class="text-xl font-semibold">{service.name}</h1>
		<p class="text-sm uppercase text-muted-foreground">{service.type}</p>
	</div>

	<div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
		<div>
			<p class="text-muted-foreground">Service ID</p>
			<p class="font-medium break-all">{service.id}</p>
		</div>
	</div>

	<div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
		<div>
			<p class="text-muted-foreground">Repository</p>
			<p class="font-medium">{service.git_repo_name || 'Not set'}</p>
		</div>
	</div>
</div>
