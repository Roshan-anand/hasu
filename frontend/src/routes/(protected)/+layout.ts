import { resolve } from '$app/paths';
import { fetchUserQuery, GetUserData } from '@/features/base';
import { redirect } from '@sveltejs/kit';
import axios from 'axios';

export async function load() {
	try {
		const { email } = GetUserData();
		if (!email) await fetchUserQuery();
	} catch (err) {
		if (axios.isAxiosError(err) && err.response?.status === 403)
			redirect(302, resolve('/register'));
		redirect(302, resolve('/login'));
	}
}
