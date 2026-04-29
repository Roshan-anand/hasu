import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import { api } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import type { LoginPayload, AuthResponse, RegisterPayload } from './type';
import { getUserState } from '@/features/global/store.svelte';

export function useLoginMutation() {
	const { setUser } = getUserState();
	return createMutation(() => ({
		mutationFn: (payload: LoginPayload) =>
			api.post<AuthResponse>('/auth/login', payload).then((res) => res.data),
		onSuccess: (data) => {
			setUser(data);
			goto(resolve('/'));
		}
	}));
}

export function useRegisterMutation() {
	const { setUser } = getUserState();
	return createMutation(() => ({
		mutationFn: (payload: RegisterPayload) =>
			api.post<AuthResponse>('/auth/register', payload).then((res) => res.data),
		onSuccess: (data) => {
			setUser(data);
			goto(resolve('/'));
		}
	}));
}
