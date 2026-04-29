import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type {
	ApiMessageRes,
	CreateServicePayload,
	CreateServiceResponse,
	GetRepoResult,
	GithubRepo,
	GitProviderOption
} from './type';
import { getServiceState } from './store.svelte';

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
	const featureState = getServiceState();

	return createMutation(() => ({
		mutationFn: async (payload: CreateServicePayload) => {
			const url = payload.type === 'app' ? '/service/app' : '/service/psql';
			return api.post<CreateServiceResponse>(url, payload.body).then((res) => res.data);
		},
		onSuccess: async (response) => {
			await featureState.afterCreateSuccess(response);
		},
		onError: (error) => {
			console.error('Error creating service:', error);
			axiosErr(error as Error, 'Failed to create service');
		}
	}));
}
