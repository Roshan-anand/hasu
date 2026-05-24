import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import { GetUserData } from '../global/query';
import type { ApiRes } from '@/types';
import type { ProjectListResponse } from './type';
import { getOrgProjectsQueryKey } from './const';

export function useGetAllProjectsQuery() {
	const { org_id } = GetUserData();
	return createQuery(() => ({
		queryKey: getOrgProjectsQueryKey(org_id),
		queryFn: async () =>
			api
				.get<ApiRes<ProjectListResponse[]>>('/project', {
					params: { org_id }
				})
				.then((res) => res.data.data),
		enabled: org_id !== ''
	}));
}
