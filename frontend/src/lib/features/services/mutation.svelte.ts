import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import type {
	CreateServicePayload,
	GetReposPayload,
	GithubRepo,
	RedeployPsqlServicePayload,
	ServiceListResponse,
	UpdateBranchDomainPayload,
	UpdateEnvPayload,
	UpdatePsqlServicePayload,
	DeleteAppServicePayload,
	DeletePsqlServicePayload,
	CreateServiceResponse,
	CreateAppServiceForm,
	CreatePsqlServiceBody
} from './type';
import type { ApiRes } from '@/types';
import { queryClient } from '@/query';
import { getInstanceServicesQueryKey } from './query.svelte';
import { getInstanceState } from '../instance';
import { normalizePathValue } from '@/utils';

export function useGetReposMutation() {
	return createMutation(() => ({
		mutationFn: async ({ provider, appId }: GetReposPayload) =>
			api
				.get<ApiRes<GithubRepo[]>>(provider.listApi, {
					params: { app_id: appId }
				})
				.then((res) => res.data.data),
		onError: (error) => axiosErr(error as Error, 'Failed to fetch repositories')
	}));
}

export function useCreateServiceMutation(getProjectName: () => string) {
	const instance = getInstanceState();

	return createMutation(() => ({
		mutationFn: async (formValue: CreateAppServiceForm) => {
			if (!instance.current.id) throw new Error('No instance selected');

			const env = formValue.env.split('\n').filter((line) => line.trim() !== '');
			const build_args = formValue.build_args.split('\n').filter((line) => line.trim() !== '');
			const build_secrets = formValue.build_secrets
				.split('\n')
				.filter((line) => line.trim() !== '');

			const payload: CreateServicePayload = {
				instance_id: instance.current.id,
				name: formValue.name.trim(),
				git_provider: formValue.git_provider,
				gh_app_id: formValue.gh_app_id,
				gh_repo_id: formValue.gh_repo_id,
				default_branch: formValue.default_branch,
				build_path: normalizePathValue(formValue.build_path),
				watch_path: normalizePathValue(formValue.watch_path),
				public: formValue.public,
				port: formValue.port,
				env,
				build_args,
				build_secrets,
				docker_build: {
					file_path: formValue.docker_build.file_path,
					context_path: formValue.docker_build.context_path,
					build_stage: formValue.docker_build.build_stage
				}
			};

			return api
				.post<ApiRes<CreateServiceResponse>>('/service/app', payload)
				.then((res) => res.data);
		},
		onSuccess: ({ data, message }) => {
			queryClient.invalidateQueries({
				queryKey: getInstanceServicesQueryKey(instance.current.id as string)
			});
			toast.success(message || 'App Service created successfully');
			goto(
				resolve('/(protected)/[project]/[service_type]/[service]?tab=deployment', {
					project: getProjectName(),
					service_type: data.type,
					service: data.name
				})
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useCreatePsqlServiceMutation(getProjectName: () => string) {
	const instance = getInstanceState();

	return createMutation(() => ({
		mutationFn: async (formValue: CreatePsqlServiceBody) => {
			if (!instance.current.id) throw new Error('No instance selected');

			const payload = {
				instance_id: instance.current.id,
				name: formValue.name.trim(),
				db_name: formValue.db_name.trim(),
				db_user: formValue.db_user.trim(),
				db_password: formValue.db_password,
				image: formValue.image.trim(),
				volume: formValue.volume
			};

			return api
				.post<ApiRes<CreateServiceResponse>>('/service/psql', payload)
				.then((res) => res.data);
		},
		onSuccess: ({ message }, { volume }) => {
			// invalidate orphan volume caches when a reattach happened
			if (volume) {
				queryClient.invalidateQueries({ queryKey: ['orphan-volumes'] });
			}
			toast.success(message || 'PSQL Service created successfully');
			goto(resolve('/(protected)/[project]', { project: getProjectName() }));
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create service')
	}));
}

export function useDeleteAppServiceMutation() {
	const instance = getInstanceState();

	return createMutation(() => ({
		mutationFn: async (payload: DeleteAppServicePayload) =>
			api.delete<ApiRes<null>>('/service/app', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			toast.success(message || 'Application deleted successfully');

			if (!instance.current.id) return;
			queryClient.setQueryData(
				getInstanceServicesQueryKey(instance.current.id),
				(cachedRows: ServiceListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== service_id);
				}
			);
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete Application')
	}));
}

export function useDeletePsqlServiceMutation() {
	const instance = getInstanceState();

	return createMutation(() => ({
		mutationFn: async (payload: DeletePsqlServicePayload) =>
			api.delete<ApiRes<null>>('/service/psql', { data: payload }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			queryClient.setQueryData(
				getInstanceServicesQueryKey(instance.current.id as string),
				(cachedRows: ServiceListResponse[] | undefined) => {
					if (!cachedRows) return [];
					return cachedRows.filter((row) => row.id !== service_id);
				}
			);
			toast.success(message || 'psql deleted successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete psql')
	}));
}

export function useUpdateBranchDomainMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdateBranchDomainPayload) =>
			api.put<ApiRes<null>>('/service/app/domain', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['branch-domain', getServiceId()]
			});
			toast.success(message || 'Domain updated successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update domain')
	}));
}

type UpdateEnvFormValues = {
	service_id: string;
	env: string;
	build_args: string;
	build_secrets: string;
};

export function useUpdateEnvMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async ({ env, build_args, build_secrets, ...rest }: UpdateEnvFormValues) => {
			const payload: UpdateEnvPayload = {
				...rest,
				env: env.split('\n').filter((l) => l.trim() !== ''),
				build_args: build_args.split('\n').filter((l) => l.trim() !== ''),
				build_secrets: build_secrets.split('\n').filter((l) => l.trim() !== '')
			};
			return api.put<ApiRes<null>>('/service/app/env', payload).then((res) => res.data);
		},
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['service-env', getServiceId()]
			});
			// TODO : show a button to rebuild / restart the service
			toast.success(message || 'Env updated successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update env')
	}));
}

export function useUpdatePsqlServiceMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdatePsqlServicePayload) =>
			api.put<ApiRes<null>>('/service/psql', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['psql-service-details', getServiceId()]
			});
			toast.success(message || 'PSQL details updated successfully');
			// TODO : show an info msg to redploy
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update PSQL details')
	}));
}

export function useRedeployPsqlServiceMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: RedeployPsqlServicePayload) =>
			api.post<ApiRes<null>>('/service/psql/redeploy', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			toast.success(message || 'PSQL redeploy started');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to redeploy PSQL service')
	}));
}
