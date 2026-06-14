import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type { ApiRes } from '@/types';
import { queryClient } from '@/query';
import type { Organization } from '@/features/auth';
import type {
	CreateProjectPayload,
	DeleteProjectPayload,
	ProjectListResponse,
	DeleteVolumePayload,
	ProjectResponse,
	SwitchOrgPayload,
	CreateOrgPayload,
	RenameOrgPayload,
	DeleteOrgPayload,
	TransferProjectPayload,
	RenameProjectPayload,
	RenameProjectResponse,
	RenameInstancePayload,
	TransferVolumePayload,
	RenameVolumePayload
} from './type';
import { getOrgProjectsQueryKey, getOrgsQueryKey } from './const';
import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import { getOrgState } from './store.svelte';
import { GetUserData } from './query.svelte';

export function useCreateProjectMutation() {
	const currentOrg = getOrgState();
	return createMutation(() => ({
		mutationFn: async (payload: CreateProjectPayload) =>
			api.post<ApiRes<ProjectResponse>>('/project', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			queryClient.setQueryData(
				getOrgProjectsQueryKey(currentOrg.id),
				(cachedRows: ProjectListResponse | undefined) => {
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
	const currentOrg = getOrgState();
	return createMutation(() => ({
		mutationFn: async (payload: DeleteProjectPayload) =>
			api.delete<ApiRes<null>>('/project', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { project_id }) => {
			queryClient.setQueryData(
				getOrgProjectsQueryKey(currentOrg.id),
				(cachedRows: ProjectListResponse | undefined) => {
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

export function useSwitchOrgMutation() {
	const { setCurrentOrg } = getOrgState();
	return createMutation(() => ({
		mutationFn: (payload: SwitchOrgPayload) =>
			api.post<ApiRes<Organization>>('/org/switch', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			setCurrentOrg(data.id, data.name);
			goto(resolve('/')); // TODO : only resolve to `/` if hash path != #/git /memeber /storage
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

export function useRenameOrgMutation() {
	const { email } = GetUserData();
	const org = getOrgState();
	return createMutation(() => ({
		mutationFn: (payload: RenameOrgPayload) =>
			api.put<ApiRes<Organization>>('/org/rename', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			queryClient.setQueryData(getOrgsQueryKey(email), (cachedOrgs: Organization[] | undefined) => {
				if (!cachedOrgs) return;
				return cachedOrgs.map((org) => (org.id === data.id ? { ...org, name: data.name } : org));
			});
			if (org.id === data.id) {
				org.setCurrentOrg(data.id, data.name);
			}
			toast.success(message || 'Organization renamed successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to rename organization')
	}));
}

export function useDeleteOrgMutation() {
	const { email } = GetUserData();
	return createMutation(() => ({
		mutationFn: (payload: DeleteOrgPayload) =>
			api.delete<ApiRes<null>>('/org', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: getOrgsQueryKey(email) });
			toast.success(message || 'Organization deleted successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to delete organization')
	}));
}

export function useTransferProjectMutation() {
	return createMutation(() => ({
		mutationFn: (payload: TransferProjectPayload) =>
			api.put<ApiRes<null>>('/project/transfer', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: ['project-list'] });
			queryClient.invalidateQueries({ queryKey: ['org-projects'] });
			toast.success(message || 'Project transferred successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to transfer project')
	}));
}
export function useRenameProjectMutation() {
	const currentOrg = getOrgState();
	return createMutation(() => ({
		mutationFn: (payload: RenameProjectPayload) =>
			api.put<ApiRes<RenameProjectResponse>>('/project/rename', payload).then((res) => res.data),
		onSuccess: ({ data, message }) => {
			queryClient.setQueryData(
				getOrgProjectsQueryKey(currentOrg.id),
				(cachedRows: ProjectListResponse | undefined) => {
					if (!cachedRows) return;
					return cachedRows.map((row) => (row.id === data.id ? { ...row, name: data.name } : row));
				}
			);
			toast.success(message || 'Project renamed successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to rename project')
	}));
}

export function useTransferVolumeMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: TransferVolumePayload) =>
			api.put<ApiRes<null>>('/org/transfer-volume', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: ['org-volumes'] });
			toast.success(message || 'Volume transferred successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to transfer volume')
	}));
}

export function useRenameInstanceMutation() {
	return createMutation(() => ({
		mutationFn: (payload: RenameInstancePayload) =>
			api.put<ApiRes<null>>('/instance/rename', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			// Invalidate all instance queries (keyed by project name)
			queryClient.invalidateQueries({ queryKey: ['all-instance'] });
			toast.success(message || 'Instance renamed successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to rename instance')
	}));
}

export function useRenameVolumeMutation() {
	return createMutation(() => ({
		mutationFn: (payload: RenameVolumePayload) =>
			api.patch<ApiRes<null>>('/volume', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: ['orphan-volumes'] });
			toast.success(message || 'Volume renamed successfully');
		},
		onError: (error) => axiosErr(error, 'Failed to rename volume')
	}));
}
