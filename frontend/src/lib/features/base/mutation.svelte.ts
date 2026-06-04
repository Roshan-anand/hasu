import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type { ApiRes } from '@/types';
import { GetUserData } from '../global/query';
import { queryClient } from '@/query';
import type { Organization } from '@/features/auth/type';
import type {
	CreateProjectPayload,
	DeleteProjectPayload,
	ProjectListResponse,
	DeleteVolumePayload
} from './type';
import { getOrgProjectsQueryKey, getOrgsQueryKey } from './const';
import { getCurrentOrgState } from '../global/store.svelte';

export function useCreateProjectMutation() {
	const currentOrg = getCurrentOrgState();
	return createMutation(() => ({
		mutationFn: async (payload: CreateProjectPayload) =>
			api.post<ApiRes<ProjectListResponse>>('/project', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			queryClient.setQueryData(
				getOrgProjectsQueryKey(currentOrg.id),
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
	const currentOrg = getCurrentOrgState();
	return createMutation(() => ({
		mutationFn: async (payload: DeleteProjectPayload) =>
			api.delete<ApiRes<null>>('/project', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { project_id }) => {
			queryClient.setQueryData(
				getOrgProjectsQueryKey(currentOrg.id),
				(cachedRows: ProjectListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== project_id);
				}
			);
			toast.success(message || 'Project deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete project')
	}));
}

export function useDeleteVolumeMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: DeleteVolumePayload) =>
			api.delete<ApiRes<null>>('/volume', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: ['orphan-volumes'] });
			toast.success(message || 'Volume deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete volume')
	}));
}

type SwitchOrgPayload = {
	org_id: string;
};

type CreateOrgPayload = {
	name: string;
};

export function useSwitchOrgMutation() {
	const { setOrg } = getCurrentOrgState();
	return createMutation(() => ({
		mutationFn: (payload: SwitchOrgPayload) =>
			api.post<ApiRes<Organization>>('/org/switch', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			setOrg(data.id, data.name);
			toast.success(message || 'Organization switched successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to switch organization')
	}));
}

export function useCreateOrgMutation() {
	const { email } = GetUserData();
	return createMutation(() => ({
		mutationFn: (payload: CreateOrgPayload) =>
			api.post<ApiRes<Organization>>('/org', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			queryClient.setQueryData(getOrgsQueryKey(email), (cachedOrgs: Organization[] | undefined) => {
				if (!cachedOrgs) return [data];
				if (cachedOrgs.some((org) => org.id === data.id)) return cachedOrgs;
				return [data, ...cachedOrgs];
			});
			toast.success(message);
		},
		onError: (error) => axiosErr(error, 'Failed to create organization')
	}));
}
