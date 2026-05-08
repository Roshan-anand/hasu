import { api } from '@/axios';
import type { AuthResponse } from '@/features/auth/type';
import { queryClient } from '@/query';

const authUserQueryKey = () => ['auth', 'user'];

// query to fetch auth user data
export const fetchUserQuery = () =>
	queryClient.fetchQuery({
		queryKey: authUserQueryKey(),
		queryFn: () => api.get<AuthResponse>('/auth/user').then((res) => res.data)
	});

// helper function to get auth user data from cache
export const GetUserData = (): AuthResponse =>
	queryClient.getQueryData<AuthResponse>(authUserQueryKey()) || {
		name: '',
		email: '',
		org_id: '',
		org_name: ''
	};

// helper function to update user organization data in cache
export const setUserCurrentOrg = (orgData: Pick<AuthResponse, 'org_id' | 'org_name'>) => {
	const currentData = queryClient.getQueryData<AuthResponse>(authUserQueryKey());
	if (!currentData) return;

	queryClient.setQueryData<AuthResponse>(authUserQueryKey(), {
		...currentData,
		org_id: orgData.org_id,
		org_name: orgData.org_name
	});
};

export const setUserData = (userData: AuthResponse | null) =>
	queryClient.setQueryData<AuthResponse | null>(authUserQueryKey(), userData);
