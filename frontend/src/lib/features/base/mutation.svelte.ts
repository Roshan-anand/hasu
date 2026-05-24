import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type { ApiRes } from '@/types';
import { GetUserData } from '../global/query';
import { queryClient } from '@/query';
import type { CreateProjectPayload, DeleteProjectPayload, ProjectListResponse } from './type';
import { getOrgProjectsQueryKey } from './const';

export function useCreateProjectMutation() {
	const { org_id } = GetUserData();
	return createMutation(() => ({
		mutationFn: async (payload: CreateProjectPayload) =>
			api.post<ApiRes<ProjectListResponse>>('/project', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			queryClient.setQueryData(
				getOrgProjectsQueryKey(org_id),
				(cachedRows: ProjectListResponse[] | undefined) => {
					if (!cachedRows) return [data];
					if (cachedRows.some((row) => row.id === data.id)) return cachedRows;
					return [data, ...cachedRows];
				}
			);
			toast.success(message || 'Project created successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create project')
	}));
}

export function useDeleteProjectMutation() {
	const { org_id } = GetUserData();
	return createMutation(() => ({
		mutationFn: async (payload: DeleteProjectPayload) =>
			api.delete<ApiRes<null>>('/project', { data: payload }).then((res) => res.data),
		onSuccess: (res, payload) => {
			queryClient.setQueryData(
				getOrgProjectsQueryKey(org_id),
				(cachedRows: ProjectListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== payload.project_id);
				}
			);
			toast.success(res.message || 'Project deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete project')
	}));
}
