import { api, axiosErr } from '@/axios';
import { queryClient } from '@/query';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import { getGithubAppsQueryKey } from './query.svelte';
import type { DeleteGithubAppPayload, GithubApp } from './type';
import { GetUserData } from '../global/query';
import type { ApiRes } from '@/types';

export function useDeleteGithubAppMutation() {
	const { org_id } = GetUserData();

	return createMutation(() => ({
		mutationFn: (payload: DeleteGithubAppPayload) =>
			api.delete<ApiRes<number>>('/provider/github/app', { data: payload }).then((res) => res.data),
		onSuccess: ({ data }) => {
			queryClient.setQueryData(
				getGithubAppsQueryKey(org_id),
				(cachedApps: GithubApp[] | null | undefined) => {
					if (!cachedApps) return null;

					const remainingApps = cachedApps.filter((app) => app.app_id !== data);
					return remainingApps.length > 0 ? remainingApps : null;
				}
			);

			toast.success('Github app deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete github app')
	}));
}
