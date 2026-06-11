import { api } from '@/axios';
import { createQuery } from '@tanstack/svelte-query';
import type { ApiRes } from '@/types';
import type { ProfileResponse } from './type';
import { profileQueryKeys } from './const';

export function useProfileQuery() {
	return createQuery(() => ({
		queryKey: profileQueryKeys.all,
		queryFn: () => api.get<ApiRes<ProfileResponse>>('/profile').then((res) => res.data.data),
		enabled: true
	}));
}
