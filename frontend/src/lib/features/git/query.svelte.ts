import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { GithubApp } from './type';
import type { ApiRes } from '@/types';
import { getBaseState } from '../global/store.svelte';

export const getGithubAppsQueryKey = (orgId: string) => ['github-apps', orgId] as const;
export function useGithubAppsQuery() {
	return createQuery(() => {
		const base = getBaseState();
		return {
			queryKey: getGithubAppsQueryKey(base.currentOrg.id),
			queryFn: () =>
				api
					.get<ApiRes<GithubApp[] | null>>('/provider/github/app/list')
					.then((res) => res.data.data),
			enabled: base.currentOrg.id != ''
		};
	});
}
