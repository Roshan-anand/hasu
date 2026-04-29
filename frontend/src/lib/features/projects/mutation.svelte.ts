import { api, axiosErr } from '@/axios';
import { queryClient } from '@/query';
import { getUserState } from '@/features/global/store.svelte';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import { getProjectsQueryKey } from './query.svelte';
import { getProjectState } from './store.svelte';
import type { ApiMessageRes, CreateProjectPayload, DeleteProjectPayload, Project } from './type';

export function useCreateProjectMutation() {
	const { closeDialog } = getProjectState();
	const { currentOrg } = getUserState();

	return createMutation(() => ({
		mutationFn: (payload: CreateProjectPayload) =>
			api.post<Project>('/project', payload).then((res) => res.data),
		onSuccess: (createdProject) => {
			queryClient.setQueryData(
				getProjectsQueryKey(currentOrg.id),
				(cachedProjects: Project[] | undefined) => {
					if (!cachedProjects) return [createdProject];
					return [createdProject, ...cachedProjects];
				}
			);

			closeDialog();
			toast.success('Project created successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Faild to create project')
	}));
}

export function useDeleteProjectMutation() {
	const featureState = getProjectState();
	const { currentOrg } = getUserState();

	return createMutation(() => ({
		mutationFn: (payload: DeleteProjectPayload) =>
			api.delete<ApiMessageRes>('/project', { data: payload }).then((res) => res.data),
		onMutate: (payload) => {
			featureState.deletingProjectId = payload.id;
		},
		onSuccess: (response, payload) => {
			queryClient.setQueryData(
				getProjectsQueryKey(currentOrg.id),
				(cachedProjects: Project[] | undefined) => {
					if (!cachedProjects) return [];
					return cachedProjects.filter((project) => project.id !== payload.id);
				}
			);
			toast.success(response.message || 'Project deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Faild to delete project'),
		onSettled: () => {
			featureState.deletingProjectId = '';
		}
	}));
}
