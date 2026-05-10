import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { GithubApp } from './type';
import { GetUserData } from '../global/query';

export const getGithubAppsQueryKey = (orgId: string) => ['github-apps', orgId] as const;
export function useGithubAppsQuery() {
	const { org_id } = GetUserData();
	return createQuery(() => ({
		queryKey: getGithubAppsQueryKey(org_id),
		queryFn: () => api.get<GithubApp[] | null>('/provider/github/app/list').then((res) => res.data),
		enabled: org_id != ''
	}));
}
