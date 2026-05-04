<script lang="ts">
	type ErrorValue = string | { message?: string } | undefined | null;

	const { errors }: { errors?: ErrorValue[] } = $props();

	function getErrorMessage(error: ErrorValue): string {
		if (!error) return '';

		if (typeof error === 'string') {
			return error;
		}

		if (typeof error === 'object' && 'message' in error) {
			return error.message ?? '';
		}

		return '';
	}
</script>

{#if errors?.length}
	{#each errors as error, i (i)}
		{#if getErrorMessage(error)}
			<em role="alert" class="text-sm font-medium text-destructive">
				{getErrorMessage(error)}
			</em>
		{/if}
	{/each}
{/if}
