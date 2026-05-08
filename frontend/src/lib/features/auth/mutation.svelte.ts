import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import { api } from '@/axios';
import { createMutation } from '@tanstack/svelte-query';
import type { LoginPayload, AuthResponse, RegisterPayload } from './type';
import { setUserData } from '../global/query';

export function useLoginMutation() {
	return createMutation(() => ({
		mutationFn: (payload: LoginPayload) =>
			api.post<AuthResponse>('/auth/login', payload).then((res) => res.data),
		onSuccess: (data) => {
			setUserData(data);
			goto(resolve('/'));
		}
	}));
}

export function useRegisterMutation() {
	return createMutation(() => ({
		mutationFn: (payload: RegisterPayload) =>
			api.post<AuthResponse>('/auth/register', payload).then((res) => res.data),
		onSuccess: (data) => {
			setUserData(data);
			goto(resolve('/'));
		}
	}));
}
