import { api, axiosErr } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import { toast } from 'svelte-sonner';
import { queryClient } from '@/query';
import type { ApiRes } from '@/types';
import type { UpdateProfilePayload, ChangePasswordPayload } from './type';
import { profileQueryKeys } from './const';

export function useUpdateProfileMutation() {
	return createMutation(() => ({
		mutationFn: (payload: UpdateProfilePayload) =>
			api.put<ApiRes<null>>('/profile', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			queryClient.invalidateQueries({ queryKey: profileQueryKeys.all });
			toast.success(message || 'Profile updated successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to update profile')
	}));
}

export function useChangePasswordMutation() {
	return createMutation(() => ({
		mutationFn: (payload: ChangePasswordPayload) =>
			api.put<ApiRes<null>>('/profile/password', payload).then((res) => res.data),
		onSuccess: ({ message }) => {
			toast.success(message || 'Password changed successfully');
		},
		onError: (error) => axiosErr(error as Error, 'Failed to change password')
	}));
}
