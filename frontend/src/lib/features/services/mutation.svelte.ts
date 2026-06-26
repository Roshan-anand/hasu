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
	UpdateServiceDomainPayload,
	UpdateEnvPayload,
	UpdatePsqlServicePayload,
	DeleteAppServicePayload,
	DeletePsqlServicePayload,
	CreateServiceResponse,
	CreateAppServiceForm,
	CreatePsqlServiceBody,
	ScaleAppServicePayload,
	CreateDependencyPayload,
	UpdateDependencyPayload,
	ServiceDependency
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
				resolve('/(protected)/[project]/[service]?tab=deployment', {
					project: getProjectName(),
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
		onSuccess: ({ message }) => {
			toast.success(message || 'Application deleted successfully');

			if (!instance.current.id) return;
			queryClient.invalidateQueries({
				queryKey: getInstanceServicesQueryKey(instance.current.id)
			});

			// eslint-disable svelte/no-navigation-without-resolve
			goto('..');
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

export function useUpdateServiceDomainMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: UpdateServiceDomainPayload) =>
			api.put<ApiRes<null>>('/service/app/domain', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['service-details', getServiceId()]
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

export function useScaleAppServiceMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async (payload: ScaleAppServicePayload) =>
			api.post<ApiRes<null>>('/service/app/scale', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: ['service-settings', getServiceId()] });
			queryClient.invalidateQueries({ queryKey: ['service-details', getServiceId()] });
			toast.success(message || 'Replicas updated');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to scale service')
	}));
}

export function usePauseAppServiceMutation() {
	return createMutation(() => ({
		mutationFn: async ({ service_id }: { service_id: string }) =>
			api.post<ApiRes<null>>('/service/app/pause', { service_id }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			queryClient.invalidateQueries({ queryKey: ['service-details', service_id] });
			toast.success(message || 'Service paused');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to pause service')
	}));
}

export function useResumeAppServiceMutation() {
	return createMutation(() => ({
		mutationFn: async ({ service_id }: { service_id: string }) =>
			api.post<ApiRes<null>>('/service/app/resume', { service_id }).then((res) => res.data),
		onSuccess: ({ message }, { service_id }) => {
			queryClient.invalidateQueries({ queryKey: ['service-details', service_id] });
			toast.success(message || 'Service resumed');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to resume service')
	}));
}

export function useStopPredefServiceMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async () =>
			api
				.post<ApiRes<null>>('/service/stop', { service_id: getServiceId() })
				.then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: ['psql-service-details', getServiceId()] });
			toast.success(message || 'Service stopped');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to stop service')
	}));
}

export function useStartPredefServiceMutation(getServiceId: () => string) {
	return createMutation(() => ({
		mutationFn: async () =>
			api
				.post<ApiRes<null>>('/service/start', { service_id: getServiceId() })
				.then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: ['psql-service-details', getServiceId()] });
			toast.success(message || 'Service started');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to start service')
	}));
}

export function useCreateDependencyMutation() {
	return createMutation(() => ({
		mutationFn: async (payload: CreateDependencyPayload) =>
			api
				.post<ApiRes<{ dependencies: ServiceDependency[] }>>('/service/app/dependencies', payload)
				.then((res) => res.data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({
				queryKey: ['service-dependencies', variables.source_service_id]
			});
			toast.success('Dependency created');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create dependency')
	}));
}

export function useDeleteDependencyMutation() {
	return createMutation(() => ({
		mutationFn: async ({ id }: { id: string; sourceServiceId: string }) =>
			api.delete<ApiRes<null>>(`/service/app/dependencies/${id}`).then((res) => res.data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({
				queryKey: ['service-dependencies', variables.sourceServiceId]
			});
			toast.success('Dependency deleted');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to delete dependency')
	}));
}

export function useUpdateDependencyMutation() {
	return createMutation(() => ({
		mutationFn: async ({ id, payload }: { id: string; payload: UpdateDependencyPayload }) =>
			api
				.put<ApiRes<ServiceDependency>>(`/service/app/dependencies/${id}`, payload)
				.then((res) => res.data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['service-dependencies'] });
			toast.success('Dependency updated');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update dependency')
	}));
}

export function useCreatePreviewMutation() {
	const instance = getInstanceState();

	return createMutation(() => ({
		mutationFn: async (payload: {
			project_id: string;
			name: string;
			pr_number: number;
			repo_id: number;
			head_branch: string;
			git_source_type: string;
			git_source_value: string;
		}) => api.post<ApiRes<null>>('/instance/preview', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({
				queryKey: ['github-prs', 'instance', instance.current.id]
			});
			toast.success(message || 'Preview creation queued');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to create preview')
	}));
}
