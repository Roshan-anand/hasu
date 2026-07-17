import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { GithubApp } from './type';
import type { ApiRes } from '@/types';
import { getOrgState } from '../base';

export const getGithubAppsQueryKey = (orgId: string) => ['github-apps', orgId] as const;
export function useGithubAppsQuery() {
	return createQuery(() => {
		const currentOrg = getOrgState();
		return {
			queryKey: getGithubAppsQueryKey(currentOrg.id),
			queryFn: () =>
				api
					.get<ApiRes<GithubApp[] | null>>('/provider/github/app/list')
					.then((res) => res.data.data),
			enabled: currentOrg.id != ''
		};
	});
}
