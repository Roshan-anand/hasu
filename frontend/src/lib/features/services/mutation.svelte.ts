import { api, axiosErr } from '@/axios';
import { queryClient } from '@/query';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type {
	ApiMessageRes,
	CreateServicePayload,
	DeleteServicePayload,
	GetRepoResult,
	GithubRepo,
	GitProviderOption,
	ServiceListResponse
} from './type';
import { getServiceState } from './store.svelte';
import { getOrgServicesQueryKey } from './query.svelte';
import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import type { ServiceType } from '@/types';
import { GetUserData } from '../global/query';

export function useGetReposMutation() {
	const featureState = getServiceState();

	return createMutation(() => ({
		mutationFn: async ({
			provider,
			appId
		}: {
			provider: GitProviderOption;
			appId: number;
		}): Promise<GetRepoResult> => {
			const response = await api.get<GithubRepo[] | ApiMessageRes>(provider.api, {
				params: { app_id: appId },
				validateStatus: (status) => status === 200 || status === 204 || status === 409
			});

			return {
				status: response.status,
				repos: response.status === 200 ? (response.data as GithubRepo[]) : [],
				message: response.status === 409 ? ((response.data as ApiMessageRes)?.message ?? '') : '',
				provider: provider.key
			};
		},
		onSuccess: (result) => {
			featureState.githubRepos = result.repos;
			if (result.status === 409) {
				toast.error(result.message || 'No github connected');
			}
		},
		onError: (error) => {
			featureState.githubRepos = [];
			axiosErr(error as Error, 'Failed to fetch repositories');
		}
	}));
}

export function useCreateServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: CreateServicePayload) =>
			api.post<string>('/service/app', payload).then((res) => res.data),
		onSuccess: (id) => {
			toast.success('Service created successfully');
			goto(
				resolve('/(protected)/(core)/[service_type]/[service_id]?tab=deployment', {
					service_type: 'app' as ServiceType,
					service_id: id
				})
			);
		},
		onError: (error) => {
			console.error('Error creating service:', error);
			axiosErr(error as Error, 'Failed to create service');
		}
	}));
}

export function useDeleteServiceMutation() {
	const { org_id } = GetUserData();

	return createMutation(() => ({
		mutationFn: async ({ service_id, type }: DeleteServicePayload) => {
			const url = type === 'psql' ? '/service/psql' : '/service/app';
			return api.delete<ApiMessageRes>(url, { data: { service_id } }).then((res) => res.data);
		},

		onSuccess: (response, payload) => {
			queryClient.setQueryData(
				getOrgServicesQueryKey(org_id),
				(cachedRows: ServiceListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== payload.service_id);
				}
			);
			toast.success(response.message || 'Service deleted successfully');
		},
		onError: (error) => {
			axiosErr(error as Error, 'Failed to delete service');
		}
	}));
}
